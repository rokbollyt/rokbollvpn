package tcputil

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
)

// Заголовок стрима для «прямого» режима.
//
// В обычном VLESS-режиме клиент просто прокидывает TCP в туннель, а SOCKS5
// рукопожатие выполняет прокси на сервере (microsocks). Из-за этого каждое
// соединение тратит два дополнительных round-trip'а СКВОЗЬ туннель, а при RTT
// туннеля в районе секунды это и есть основная задержка открытия страниц.
//
// В прямом режиме клиент сам отвечает на SOCKS5 (локально, мгновенно), а адрес
// цели передаёт сервером первым сообщением стрима. Сервер подключается к цели
// напрямую — лишние round-trip'ы исчезают, microsocks больше не нужен.
//
// Формат: [1 байт длина хоста][хост][2 байта порт, big-endian]

const maxHostLen = 255

// WriteStreamTarget пишет адрес цели в начало стрима.
func WriteStreamTarget(w io.Writer, host string, port int) error {
	if len(host) == 0 || len(host) > maxHostLen {
		return fmt.Errorf("некорректная длина хоста: %d", len(host))
	}
	buf := make([]byte, 0, 1+len(host)+2)
	buf = append(buf, byte(len(host)))
	buf = append(buf, host...)
	buf = append(buf, byte(port>>8), byte(port))
	_, err := w.Write(buf)
	return err
}

// ReadStreamTarget читает адрес цели из начала стрима.
func ReadStreamTarget(r io.Reader) (string, error) {
	var l [1]byte
	if _, err := io.ReadFull(r, l[:]); err != nil {
		return "", err
	}
	if l[0] == 0 {
		return "", fmt.Errorf("пустой хост в заголовке стрима")
	}
	host := make([]byte, int(l[0]))
	if _, err := io.ReadFull(r, host); err != nil {
		return "", err
	}
	var p [2]byte
	if _, err := io.ReadFull(r, p[:]); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(p[:])
	if port == 0 {
		return "", fmt.Errorf("нулевой порт в заголовке стрима")
	}
	return net.JoinHostPort(string(host), strconv.Itoa(int(port))), nil
}
