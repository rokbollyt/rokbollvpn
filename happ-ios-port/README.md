# HappPort (iOS Native Port with MAX-TURN & Hysteria2)

Полная реализация клиента Happ для iOS с поддержкой **TrollStore** (jailbroken / trollstore iOS 14.0–17.0+).

## Архитектура

1. **UI Layer (`HappPort`)**:
   - Написан на SwiftUI.
   - Главный экран с большой кнопкой подключения (Power), индикатором статуса, переключателем режима «MAX-TURN Relay» (обход DPI через инфраструктуру звонков VK/MAX) и выбором сервера.
   - Импорт подписок и ключей (`hysteria2://`, `vless://`, `vmess://`, `ss://`).
   - Настройки сессии MAX (Call Token, Device ID, Login Token).

2. **Network Extension (`HappPortTunnel`)**:
   - `NEPacketTunnelProvider` — полноценный системный VPN на уровне устройства (в статус-баре iOS загорается значок `[VPN]`).
   - Исключающие маршруты (bypass) для диапазонов OK/VK/MAX и DNS-резолверов, предотвращающие рекурсивную петлю туннеля.
   - Цикл передачи L3 IP-пакетов через CGO-мост в Go-ядро (`RouterWritePacket` / `RouterReadPacket`).

3. **Core Engine (`go-core`, `librouter.a`)**:
   - Написан на Go, собирается в `c-archive`.
   - **MAX-TURN Relay**: прозрачный UDP-туннель через TURN-сервера звонков MAX (`RunRawUDPRelay`).
   - **Hysteria 2 Client**: бинарник Hysteria2 darwin/arm64, упакованный в Bundle приложения, направленный на локальный порт релея.
   - **gVisor Netstack (`tun2socks`)**: виртуальный стек TCP/IP в юзерспейсе, преобразующий системный IP-трафик устройства в локальный SOCKS5 прокси Hysteria2.

## Сборка IPA через GitHub Actions

Так как кросс-компиляция Xcode проекта для iOS с расширением `NEPacketTunnelProvider` требует macOS и компилятора clang со средой iPhoneOS SDK, проект подготовлен для сборки в один клик через GitHub Actions:

1. Запушьте репозиторий на GitHub.
2. Перейдите во вкладку **Actions** -> выберите **Build TrollStore IPA** -> нажмите **Run workflow**.
3. Дождитесь завершения сборки (~3-4 минуты). В артефактах сборки появится готовый файл `HappPort-TrollStore.ipa`.
4. Скачайте `.ipa` на iPhone и откройте через **TrollStore** -> нажмите **Install**.

