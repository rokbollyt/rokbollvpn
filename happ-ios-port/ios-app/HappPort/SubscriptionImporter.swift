import Foundation

/// ProxyServer represents a configured VPN server across any supported protocol:
/// - AmneziaWG (AWG)
/// - WireGuard (WG)
/// - Hysteria 2 (HY2)
/// - OpenVPN UDP (OVPN)
struct ProxyServer: Identifiable, Codable, Equatable {
    var id: String = UUID().uuidString
    var name: String
    var protocolType: String // "AmneziaWG", "WireGuard", "Hysteria 2", "OpenVPN"
    var peerAddr: String     // host:port (UDP)
    var uri: String          // Original or generated share URI

    // WireGuard & AmneziaWG
    var privateKey: String = ""
    var publicKey: String = ""
    var presharedKey: String = ""
    var clientIP: String = "10.0.0.2"
    var dns: String = "1.1.1.1"
    var mtu: Int = 1420

    // AmneziaWG Obfuscation
    var jc: Int = 0
    var jmin: Int = 0
    var jmax: Int = 0
    var s1: Int = 0
    var s2: Int = 0
    var h1: Int = 0
    var h2: Int = 0
    var h3: Int = 0
    var h4: Int = 0

    // OpenVPN
    var openvpnConfig: String = ""
    var username: String = ""
    var password: String = ""

    // Hysteria 2
    var hysteriaURI: String = ""

    /// Generates a compact, 1-line share link for Telegram or copying.
    func toShareLink() -> String {
        switch protocolType {
        case "AmneziaWG":
            var link = "awg://\(privateKey)@\(peerAddr)?ip=\(clientIP)&pub=\(publicKey)"
            if jc > 0 || h1 > 0 {
                link += "&jc=\(jc)&jmin=\(jmin)&jmax=\(jmax)&s1=\(s1)&s2=\(s2)&h1=\(h1)&h2=\(h2)&h3=\(h3)&h4=\(h4)"
            }
            if !presharedKey.isEmpty { link += "&psk=\(presharedKey)" }
            if !dns.isEmpty { link += "&dns=\(dns)" }
            return link + "#" + (name.addingPercentEncoding(withAllowedCharacters: .urlFragmentAllowed) ?? name)

        case "WireGuard":
            var link = "wg://\(privateKey)@\(peerAddr)?ip=\(clientIP)&pub=\(publicKey)"
            if !presharedKey.isEmpty { link += "&psk=\(presharedKey)" }
            if !dns.isEmpty { link += "&dns=\(dns)" }
            return link + "#" + (name.addingPercentEncoding(withAllowedCharacters: .urlFragmentAllowed) ?? name)

        case "OpenVPN":
            if !openvpnConfig.isEmpty {
                let b64 = Data(openvpnConfig.utf8).base64EncodedString()
                return "ovpn://\(b64)#" + (name.addingPercentEncoding(withAllowedCharacters: .urlFragmentAllowed) ?? name)
            }
            return "ovpn://\(username):\(password)@\(peerAddr)?proto=udp#" + (name.addingPercentEncoding(withAllowedCharacters: .urlFragmentAllowed) ?? name)

        case "Hysteria 2":
            return !hysteriaURI.isEmpty ? hysteriaURI : uri

        default:
            return uri
        }
    }
}

@MainActor
final class ServerManager: ObservableObject {
    @Published var servers: [ProxyServer] = []
    @Published var selectedServerId: String = ""

    private let storageKey = "HappPort_Servers_v2"
    private let selectedKey = "HappPort_SelectedServer_v2"

    init() {
        loadServers()
        if servers.isEmpty {
            // Default sample servers
            let defaultHy2 = "hysteria2://02df03e3-3029-4c95-bffe-8f211b480672@edge.hankli1214.xyz:443/?sni=hank.hankli1214.xyz&obfs=salamander&obfs-password=MeJA9qgoYMH4UMunHowisrbd&upmbps=50&downmbps=500#Japan (MAX HY2)"
            importFromText(defaultHy2)

            let sampleAwg = "awg://aGFwcC1wcml2YXRlLWtleS0zMndn...==@185.195.23.44:51820?ip=10.8.0.2&pub=c2VydmVyLXB1YmtleS0zMndn...==&jc=4&jmin=40&jmax=70&s1=15&s2=20&h1=1&h2=2&h3=3&h4=4#AmneziaWG Sample"
            importFromText(sampleAwg)
        }
    }

    var selectedServer: ProxyServer? {
        servers.first { $0.id == selectedServerId } ?? servers.first
    }

    func addServer(_ server: ProxyServer) {
        servers.append(server)
        if selectedServerId.isEmpty {
            selectedServerId = server.id
        }
        saveServers()
    }

    func removeServer(at offsets: IndexSet) {
        servers.remove(atOffsets: offsets)
        if !servers.contains(where: { $0.id == selectedServerId }) {
            selectedServerId = servers.first?.id ?? ""
        }
        saveServers()
    }

    /// Imports from any supported format:
    /// - 1-line URLs: awg://, wg://, ovpn://, hysteria2://, hy2://, happ://, max://
    /// - Raw config files: WireGuard/AmneziaWG .conf, OpenVPN .ovpn
    func importFromText(_ input: String) {
        let trimmed = input.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }

        // Check if raw WireGuard / AmneziaWG configuration text
        if trimmed.contains("[Interface]") && trimmed.contains("[Peer]") {
            if let server = parseWireGuardConf(trimmed) {
                addServer(server)
                return
            }
        }

        // Check if raw OpenVPN configuration text
        if (trimmed.contains("client") || trimmed.contains("dev tun")) && trimmed.contains("remote ") {
            if let server = parseOpenVPNText(trimmed) {
                addServer(server)
                return
            }
        }

        // Split by lines in case multiple URIs are pasted
        let lines = trimmed.components(separatedBy: .newlines)
        for line in lines {
            let item = line.trimmingCharacters(in: .whitespacesAndNewlines)
            guard !item.isEmpty else { continue }

            if item.hasPrefix("awg://") || item.hasPrefix("amnezia://") {
                if let server = parseAmneziaURI(item) { addServer(server) }
            } else if item.hasPrefix("wg://") || item.hasPrefix("wireguard://") {
                if let server = parseWireGuardURI(item) { addServer(server) }
            } else if item.hasPrefix("ovpn://") || item.hasPrefix("openvpn://") {
                if let server = parseOpenVPNURI(item) { addServer(server) }
            } else if item.hasPrefix("hysteria2://") || item.hasPrefix("hy2://") {
                let name = extractFragment(from: item) ?? "Hysteria 2 Server"
                let peer = extractHostPort(from: item) ?? ""
                let s = ProxyServer(name: name, protocolType: "Hysteria 2", peerAddr: peer, uri: item, hysteriaURI: item)
                addServer(s)
            } else if item.hasPrefix("happ://") || item.hasPrefix("max://") {
                if let server = parseUnifiedJSON(item) { addServer(server) }
            }
        }
    }

    func importFromClipboardOrText(_ input: String) {
        importFromText(input)
    }

    // MARK: - Key Generation Helper

    /// Generates a random WireGuard / AmneziaWG keypair and obfuscation params
    static func generateRandomAwgKey(serverEndpoint: String, serverPubKey: String, clientIP: String = "10.8.0.2") -> ProxyServer {
        var privBytes = [UInt8](repeating: 0, count: 32)
        _ = SecRandomCopyBytes(kSecRandomDefault, 32, &privBytes)
        // Clamp private key for curve25519
        privBytes[0] &= 248
        privBytes[31] = (privBytes[31] & 127) | 64
        let privKey = Data(privBytes).base64EncodedString()

        // Generate random obfuscation parameters for AmneziaWG
        let jc = Int.random(in: 3...7)
        let jmin = Int.random(in: 40...60)
        let jmax = Int.random(in: 70...120)
        let s1 = Int.random(in: 15...35)
        let s2 = Int.random(in: 20...45)
        let h1 = Int.random(in: 100000...999999999)
        let h2 = Int.random(in: 100000...999999999)
        let h3 = Int.random(in: 100000...999999999)
        let h4 = Int.random(in: 100000...999999999)

        var server = ProxyServer(
            name: "AmneziaWG Generated",
            protocolType: "AmneziaWG",
            peerAddr: serverEndpoint.isEmpty ? "1.2.3.4:51820" : serverEndpoint,
            uri: "",
            privateKey: privKey,
            publicKey: serverPubKey.isEmpty ? "PUBKEY_PLACEHOLDER==" : serverPubKey,
            clientIP: clientIP,
            dns: "1.1.1.1",
            mtu: 1420,
            jc: jc, jmin: jmin, jmax: jmax, s1: s1, s2: s2, h1: h1, h2: h2, h3: h3, h4: h4
        )
        server.uri = server.toShareLink()
        return server
    }

    // MARK: - Parsers

    private func parseAmneziaURI(_ uri: String) -> ProxyServer? {
        // awg://<privateKey>@<host>:<port>?ip=<clientIP>&pub=<publicKey>&jc=...#Name
        let clean = uri.replacingOccurrences(of: "amnezia://", with: "").replacingOccurrences(of: "awg://", with: "")
        let name = extractFragment(from: uri) ?? "AmneziaWG Server"

        var mainPart = clean
        if let hashIdx = mainPart.firstIndex(of: "#") {
            mainPart = String(mainPart[..<hashIdx])
        }

        let queryParams = parseQueryParams(from: mainPart)
        let hostAuth = mainPart.components(separatedBy: "?").first ?? mainPart

        var privKey = ""
        var peerAddr = ""
        if hostAuth.contains("@") {
            let parts = hostAuth.components(separatedBy: "@")
            privKey = parts[0]
            peerAddr = parts[1]
        } else {
            peerAddr = hostAuth
        }

        return ProxyServer(
            name: name,
            protocolType: "AmneziaWG",
            peerAddr: peerAddr,
            uri: uri,
            privateKey: queryParams["priv"] ?? privKey,
            publicKey: queryParams["pub"] ?? queryParams["public_key"] ?? "",
            presharedKey: queryParams["psk"] ?? "",
            clientIP: queryParams["ip"] ?? queryParams["address"] ?? "10.0.0.2",
            dns: queryParams["dns"] ?? "1.1.1.1",
            mtu: Int(queryParams["mtu"] ?? "1420") ?? 1420,
            jc: Int(queryParams["jc"] ?? "4") ?? 4,
            jmin: Int(queryParams["jmin"] ?? "40") ?? 40,
            jmax: Int(queryParams["jmax"] ?? "70") ?? 70,
            s1: Int(queryParams["s1"] ?? "15") ?? 15,
            s2: Int(queryParams["s2"] ?? "20") ?? 20,
            h1: Int(queryParams["h1"] ?? "1") ?? 1,
            h2: Int(queryParams["h2"] ?? "2") ?? 2,
            h3: Int(queryParams["h3"] ?? "3") ?? 3,
            h4: Int(queryParams["h4"] ?? "4") ?? 4
        )
    }

    private func parseWireGuardURI(_ uri: String) -> ProxyServer? {
        // wg://<privateKey>@<host>:<port>?ip=<clientIP>&pub=<publicKey>#Name
        let clean = uri.replacingOccurrences(of: "wireguard://", with: "").replacingOccurrences(of: "wg://", with: "")
        let name = extractFragment(from: uri) ?? "WireGuard Server"

        var mainPart = clean
        if let hashIdx = mainPart.firstIndex(of: "#") {
            mainPart = String(mainPart[..<hashIdx])
        }

        let queryParams = parseQueryParams(from: mainPart)
        let hostAuth = mainPart.components(separatedBy: "?").first ?? mainPart

        var privKey = ""
        var peerAddr = ""
        if hostAuth.contains("@") {
            let parts = hostAuth.components(separatedBy: "@")
            privKey = parts[0]
            peerAddr = parts[1]
        } else {
            peerAddr = hostAuth
        }

        return ProxyServer(
            name: name,
            protocolType: "WireGuard",
            peerAddr: peerAddr,
            uri: uri,
            privateKey: queryParams["priv"] ?? privKey,
            publicKey: queryParams["pub"] ?? queryParams["public_key"] ?? "",
            presharedKey: queryParams["psk"] ?? "",
            clientIP: queryParams["ip"] ?? queryParams["address"] ?? "10.0.0.2",
            dns: queryParams["dns"] ?? "1.1.1.1",
            mtu: Int(queryParams["mtu"] ?? "1420") ?? 1420
        )
    }

    private func parseOpenVPNURI(_ uri: String) -> ProxyServer? {
        let clean = uri.replacingOccurrences(of: "openvpn://", with: "").replacingOccurrences(of: "ovpn://", with: "")
        let name = extractFragment(from: uri) ?? "OpenVPN UDP"

        var mainPart = clean
        if let hashIdx = mainPart.firstIndex(of: "#") {
            mainPart = String(mainPart[..<hashIdx])
        }

        // Check if payload is base64 encoded .ovpn configuration
        if let data = Data(base64Encoded: mainPart), let ovpnText = String(data: data, encoding: .utf8) {
            let peer = extractRemoteFromOvpn(ovpnText) ?? "127.0.0.1:1194"
            return ProxyServer(
                name: name,
                protocolType: "OpenVPN",
                peerAddr: peer,
                uri: uri,
                openvpnConfig: ovpnText
            )
        }

        // Otherwise parse as URL: user:pass@host:port
        let hostAuth = mainPart.components(separatedBy: "?").first ?? mainPart
        var user = ""
        var pass = ""
        var peerAddr = hostAuth

        if hostAuth.contains("@") {
            let parts = hostAuth.components(separatedBy: "@")
            let creds = parts[0].components(separatedBy: ":")
            user = creds.first ?? ""
            if creds.count > 1 { pass = creds[1] }
            peerAddr = parts[1]
        }

        return ProxyServer(
            name: name,
            protocolType: "OpenVPN",
            peerAddr: peerAddr,
            uri: uri,
            username: user,
            password: pass
        )
    }

    private func parseWireGuardConf(_ text: String) -> ProxyServer? {
        var privKey = ""
        var address = "10.0.0.2"
        var dns = "1.1.1.1"
        var pubKey = ""
        var endpoint = ""
        var psk = ""
        var jc = 0, jmin = 0, jmax = 0, s1 = 0, s2 = 0, h1 = 0, h2 = 0, h3 = 0, h4 = 0

        for line in text.components(separatedBy: .newlines) {
            let l = line.trimmingCharacters(in: .whitespaces)
            guard !l.hasPrefix("#") && l.contains("=") else { continue }
            let parts = l.components(separatedBy: "=")
            let key = parts[0].trimmingCharacters(in: .whitespaces).lowercased()
            let val = parts[1].trimmingCharacters(in: .whitespaces)

            switch key {
            case "privatekey": privKey = val
            case "address": address = val.components(separatedBy: "/").first ?? val
            case "dns": dns = val
            case "publickey": pubKey = val
            case "endpoint": endpoint = val
            case "presharedkey": psk = val
            case "jc": jc = Int(val) ?? 0
            case "jmin": jmin = Int(val) ?? 0
            case "jmax": jmax = Int(val) ?? 0
            case "s1": s1 = Int(val) ?? 0
            case "s2": s2 = Int(val) ?? 0
            case "h1": h1 = Int(val) ?? 0
            case "h2": h2 = Int(val) ?? 0
            case "h3": h3 = Int(val) ?? 0
            case "h4": h4 = Int(val) ?? 0
            default: break
            }
        }

        guard !privKey.isEmpty && !endpoint.isEmpty else { return nil }

        let isAmnezia = jc > 0 || h1 > 0
        let proto = isAmnezia ? "AmneziaWG" : "WireGuard"
        let name = "\(proto) (\(endpoint))"

        return ProxyServer(
            name: name,
            protocolType: proto,
            peerAddr: endpoint,
            uri: "",
            privateKey: privKey,
            publicKey: pubKey,
            presharedKey: psk,
            clientIP: address,
            dns: dns,
            jc: jc, jmin: jmin, jmax: jmax, s1: s1, s2: s2, h1: h1, h2: h2, h3: h3, h4: h4
        )
    }

    private func parseOpenVPNText(_ text: String) -> ProxyServer? {
        guard let remote = extractRemoteFromOvpn(text) else { return nil }
        return ProxyServer(
            name: "OpenVPN UDP (\(remote))",
            protocolType: "OpenVPN",
            peerAddr: remote,
            uri: "",
            openvpnConfig: text
        )
    }

    private func parseUnifiedJSON(_ str: String) -> ProxyServer? {
        let raw = str.replacingOccurrences(of: "happ://", with: "").replacingOccurrences(of: "max://", with: "")
        guard let data = Data(base64Encoded: raw) else { return nil }
        return try? JSONDecoder().decode(ProxyServer.self, from: data)
    }

    private func extractRemoteFromOvpn(_ text: String) -> String? {
        for line in text.components(separatedBy: .newlines) {
            let l = line.trimmingCharacters(in: .whitespaces)
            if l.hasPrefix("remote ") {
                let parts = l.components(separatedBy: .whitespaces).filter { !$0.isEmpty }
                if parts.count >= 3 {
                    return "\(parts[1]):\(parts[2])"
                } else if parts.count == 2 {
                    return "\(parts[1]):1194"
                }
            }
        }
        return nil
    }

    private func parseQueryParams(from s: String) -> [String: String] {
        var map: [String: String] = [:]
        guard let queryIdx = s.firstIndex(of: "?") else { return map }
        let queryString = String(s[s.index(after: queryIdx)...])
        for pair in queryString.components(separatedBy: "&") {
            let kv = pair.components(separatedBy: "=")
            if kv.count == 2 {
                map[kv[0].lowercased()] = kv[1].removingPercentEncoding ?? kv[1]
            }
        }
        return map
    }

    private func extractFragment(from uri: String) -> String? {
        if let fragment = uri.components(separatedBy: "#").last, !fragment.isEmpty, fragment != uri {
            return fragment.removingPercentEncoding ?? fragment
        }
        return nil
    }

    private func extractHostPort(from uri: String) -> String? {
        var s = uri
        if let prefixIdx = s.range(of: "://") {
            s = String(s[prefixIdx.upperBound...])
        }
        if let atIdx = s.range(of: "@") {
            s = String(s[atIdx.upperBound...])
        }
        if let queryIdx = s.firstIndex(of: "?") {
            s = String(s[..<queryIdx])
        }
        if let hashIdx = s.firstIndex(of: "#") {
            s = String(s[..<hashIdx])
        }
        s = s.trimmingCharacters(in: CharacterSet(charactersIn: "/ "))
        return s.isEmpty ? nil : s
    }

    private func saveServers() {
        if let data = try? JSONEncoder().encode(servers) {
            UserDefaults.standard.set(data, forKey: storageKey)
        }
        UserDefaults.standard.set(selectedServerId, forKey: selectedKey)
    }

    private func loadServers() {
        if let data = UserDefaults.standard.data(forKey: storageKey),
           let list = try? JSONDecoder().decode([ProxyServer].self, from: data) {
            self.servers = list
        }
        self.selectedServerId = UserDefaults.standard.string(forKey: selectedKey) ?? ""
    }
}
