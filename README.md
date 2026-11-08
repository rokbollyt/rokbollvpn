# ⚡ Rokboll VPN

> **Кроссплатформенный VPN-клиент нового поколения для обхода любых блокировок ТСПУ и DPI.**  
> Трафик инкапсулируется в протоколы доверенных белых сервисов (видеозвонки **MAX / VK / OK**) и игровую маскировку (**Minewire**). Провайдер видит только легитимный медиапоток DTLS/WebRTC на IP-адреса российских телеком-платформ.

---

## 📥 Загрузки (Актуальные сборки)

Все свежие релизы автоматически собираются, подписываются и публикуются в [GitHub Releases](https://github.com/rokbollyt/rokbollvpn/releases/tag/latest):

| Платформа | Ссылка | Тип установки |
|---|---|---|
| **Android** | [**Happ-MAX-Release.apk**](https://github.com/rokbollyt/rokbollvpn/releases/download/latest/Happ-MAX-Release.apk) | Прямой APK (ARM64/ARMv7), подписан |
| **iOS** | [**HappPort-TrollStore.ipa**](https://github.com/rokbollyt/rokbollvpn/releases/download/latest/HappPort-TrollStore.ipa) | TrollStore / AltStore / Sideload IPA |
| **Windows** | [**rokboll-vpn-gui.exe**](https://github.com/rokbollyt/rokbollvpn/releases/tag/latest) | Portable EXE (требуются права администратора) |

---

## 🛡️ Архитектура обхода блокировок

Обычные VPN-протоколы (WireGuard, OpenVPN, чистый Shadowsocks) распознаются и мгновенно глушатся DPI / ТСПУ по сигнатурам рукопожатия и энтропии пакетов.

В **Rokboll VPN** весь исходящий трафик оборачивается в доверенные протоколы:

```
[Твой девайс]
      │
      │  (Легитимный трафик звонка / DTLS)
      ▼
[TURN-сервер MAX / VK] ─── (Белый список ТСПУ)
      │
      ▼
[Наш VPS Сервер] ─── (Расшифровка и выход в сеть)
      │
      ▼
 [Свободный Интернет]
```

### Поддерживаемые протоколы и режимы

1. **MAX-TURN (Основной)**
   - Проброс трафика через инфраструктуру видеозвонков мессенджера MAX (VK / OK / Mail.ru).
   - Для ТСПУ это обычный видеозвонок на российские IP-адреса.
2. **VK-TURN**
   - Альтернативный звонковый туннель через TURN-кластер ВКонтакте.
3. **AmneziaWG & WireGuard поверх MAX-TURN**
   - Полная поддержка маскировки заголовков AmneziaWG (`H1`–`H4`, `Jc`, `Jmin`, `Jmax`, `S1`, `S2`) с дополнительным сокрытием внутри звонкового DTLS-релея.
4. **Minewire**
   - Маскировка туннеля под TCP/UDP сетевой трафик игрового сервера Minecraft.
5. **XRay Outbound Interceptor (Happ Mod)**
   - Автоматический перехват исходящего UDP-трафика любых стандартных протоколов (VLESS, VMess, Trojan, Hysteria 2, Shadowsocks) и перенаправление через защищённый туннель `127.0.0.1:11590`.

---

## 🔑 Форматы ключей и импорт

Клиент поддерживает **мгновенный импорт** любых конфигураций через буфер обмена (`+` $\rightarrow$ **«Из буфера обмена»**):

### 1. AmneziaWG / WireGuard URI (однострочный)

```text
awg://<CLIENT_PRIVATE_KEY>@<SERVER_IP>:<PORT>?ip=10.0.0.2/32&pub=<SERVER_PUBLIC_KEY>&dns=1.1.1.1&jc=4&jmin=40&jmax=70&s1=0&s2=0&h1=1&h2=2&h3=3&h4=4#AmneziaWG-Node
```

### 2. Сырой INI-конфиг WireGuard / AmneziaWG

```ini
[Interface]
PrivateKey = <CLIENT_PRIVATE_KEY>
Address = 10.0.0.2/32
DNS = 1.1.1.1, 8.8.8.8
Jc = 4
Jmin = 40
Jmax = 70
S1 = 0
S2 = 0
H1 = 1
H2 = 2
H3 = 3
H4 = 4

[Peer]
PublicKey = <SERVER_PUBLIC_KEY>
Endpoint = <SERVER_IP>:<PORT>
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
```

### 3. Кастомные URI протоколов

- **Minewire**: `minewire://password@<SERVER_IP>:<PORT>?mode=fast#ServerName`
- **MAX-TURN**: `max-turn://<SERVER_IP>:<PORT>?link=<CALL_LINK>&streams=4#MAX-Node`
- **VK-TURN**: `vk-turn://<SERVER_IP>:<PORT>?link=<CALL_LINK>#VK-Node`

---

## ⚙️ Установка и использование

### Android (Happ Mod)
1. Скачай [Happ-MAX-Release.apk](https://github.com/rokbollyt/rokbollvpn/releases/download/latest/Happ-MAX-Release.apk) и установи на устройство.
2. Скопируй любой ключ в буфер обмена.
3. В приложении нажми `+` $\rightarrow$ **«Из буфера обмена»**.
4. Профиль мгновенно появится в списке — нажми подключение.
5. *(Все назойливые всплывающие окна апдейтов и проверки версий вырезаны из приложения).*

### iOS (TrollStore / Sideload)
1. Скачай [HappPort-TrollStore.ipa](https://github.com/rokbollyt/rokbollvpn/releases/download/latest/HappPort-TrollStore.ipa).
2. Открой через TrollStore или установи через AltStore/SideStore.
3. Добавь профиль и подключись.

### Windows
1. Скачай `rokboll-vpn-gui.exe`.
2. Запусти от имени администратора (необходимо для развертывания Wintun-адаптера).
3. Нажми большую кнопку **ON**.

---

## 🛠️ Сборка из исходников

### Требования
- Go 1.22+
- Android NDK & SDK (для сборки native-библиотек)
- Java 17+ & Apktool 2.10.0+
- Python 3.10+

### Сборка туннельного демона (`librokboll.so`)
```powershell
cd minewire-clientSlide/cmd/rokboll-tunnel
go build -ldflags="-s -w" -o ../../../librokboll.exe .
```

### Сборка и подпись Android APK
```powershell
# 1. Сборка Java-моста и smali
python build_rokboll_smali.py

# 2. Сборка через apktool
java -jar apktool.jar b Happ_src -o Happ-unaligned.apk

# 3. Zipalign и подпись apksigner
python sign_and_release.py
```

---

## 📄 Лицензия

Проект распространяется для личного использования и исследований в области устойчивости сетевых протоколов.
