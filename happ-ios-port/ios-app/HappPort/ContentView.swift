import SwiftUI
import UIKit

struct ContentView: View {
    @StateObject private var vpn = VPNController()
    @StateObject private var serverManager = ServerManager()

    @AppStorage("useMaxTurn") private var useMaxTurn: Bool = true
    @AppStorage("maxCallToken") private var maxCallToken: String = "$laipx1vh6JVC3FdLMaGIhnRqikbMifOXTGwUUo7MNsVbuyy1grABxPinzsb0DNn8Vw36hdYaO3"
    @AppStorage("maxDeviceId") private var maxDeviceId: String = "68470aa6-34a4-49ad-bf2f-2a534624270b"
    @AppStorage("maxLoginToken") private var maxLoginToken: String = ""

    @State private var showServerList = false
    @State private var showImportSheet = false
    @State private var showGeneratorSheet = false
    @State private var showSettingsSheet = false
    @State private var importText = ""
    @State private var copiedNotice = false

    var body: some View {
        NavigationView {
            ZStack {
                Color(UIColor.systemGroupedBackground)
                    .ignoresSafeArea()

                VStack(spacing: 24) {
                    // Status Badge
                    HStack(spacing: 8) {
                        Circle()
                            .fill(vpn.isConnected ? Color.green : (vpn.isConnecting ? Color.orange : Color.gray))
                            .frame(width: 10, height: 10)
                        Text(vpn.status)
                            .font(.system(size: 15, weight: .medium))
                            .foregroundColor(.secondary)

                        if vpn.isConnected, let s = serverManager.selectedServer {
                            ProtocolBadge(proto: s.protocolType)
                        }
                    }
                    .padding(.top, 10)

                    Spacer()

                    // Big Power Button
                    Button(action: toggleConnection) {
                        ZStack {
                            Circle()
                                .fill(vpn.isConnected ?
                                      LinearGradient(colors: [Color.green, Color.mint], startPoint: .topLeading, endPoint: .bottomTrailing) :
                                      LinearGradient(colors: [Color.blue, Color.cyan], startPoint: .topLeading, endPoint: .bottomTrailing))
                                .frame(width: 160, height: 160)
                                .shadow(color: vpn.isConnected ? Color.green.opacity(0.4) : Color.blue.opacity(0.3), radius: 20, x: 0, y: 10)

                            Image(systemName: "power")
                                .font(.system(size: 64, weight: .semibold))
                                .foregroundColor(.white)
                        }
                    }
                    .disabled(vpn.isConnecting)

                    Spacer()

                    // MAX-TURN Relay Toggle Card
                    VStack(alignment: .leading, spacing: 12) {
                        Toggle(isOn: $useMaxTurn) {
                            HStack {
                                Image(systemName: "shield.righthalf.filled")
                                    .foregroundColor(.blue)
                                    .font(.system(size: 22))
                                VStack(alignment: .leading, spacing: 2) {
                                    HStack {
                                        Text("MAX-TURN Relay")
                                            .font(.system(size: 16, weight: .semibold))
                                        Text("VK / MAX")
                                            .font(.system(size: 10, weight: .bold))
                                            .padding(.horizontal, 5)
                                            .padding(.vertical, 1)
                                            .background(Color.blue.opacity(0.15))
                                            .foregroundColor(.blue)
                                            .cornerRadius(4)
                                    }
                                    Text("Wraps UDP (AWG, WG, HY2, OVPN) into video-call datagrams to bypass DPI")
                                        .font(.system(size: 11))
                                        .foregroundColor(.secondary)
                                }
                            }
                        }
                    }
                    .padding()
                    .background(Color(UIColor.secondarySystemGroupedBackground))
                    .cornerRadius(14)
                    .padding(.horizontal)

                    // Selected Server Card
                    Button(action: { showServerList = true }) {
                        HStack(spacing: 14) {
                            Image(systemName: "server.rack")
                                .font(.system(size: 22))
                                .foregroundColor(.blue)

                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text(serverManager.selectedServer?.name ?? "Select Server")
                                        .font(.system(size: 16, weight: .semibold))
                                        .foregroundColor(.primary)
                                        .lineLimit(1)
                                    if let s = serverManager.selectedServer {
                                        ProtocolBadge(proto: s.protocolType)
                                    }
                                }

                                Text(serverManager.selectedServer?.peerAddr ?? "No node selected")
                                    .font(.system(size: 12))
                                    .foregroundColor(.secondary)
                                    .lineLimit(1)
                            }

                            Spacer()

                            Image(systemName: "chevron.right")
                                .foregroundColor(.secondary)
                        }
                        .padding()
                        .background(Color(UIColor.secondarySystemGroupedBackground))
                        .cornerRadius(14)
                    }
                    .padding(.horizontal)
                    .padding(.bottom, 20)
                }
            }
            .navigationTitle("Happ")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button(action: { showSettingsSheet = true }) {
                        Image(systemName: "gearshape")
                    }
                }
                ToolbarItemGroup(placement: .navigationBarTrailing) {
                    Button(action: { showGeneratorSheet = true }) {
                        Image(systemName: "key")
                    }
                    Button(action: { showImportSheet = true }) {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(isPresented: $showServerList) {
                serverListSheet
            }
            .sheet(isPresented: $showImportSheet) {
                importSheet
            }
            .sheet(isPresented: $showGeneratorSheet) {
                KeyGeneratorSheet(serverManager: serverManager)
            }
            .sheet(isPresented: $showSettingsSheet) {
                settingsSheet
            }
        }
    }

    private func toggleConnection() {
        if vpn.isConnected || vpn.isConnecting {
            vpn.stop()
        } else {
            guard let server = serverManager.selectedServer else { return }

            var proto = "hysteria2"
            let pt = server.protocolType.lowercased()
            if pt.contains("amnezia") || pt.contains("awg") {
                proto = "amneziawg"
            } else if pt.contains("wireguard") || pt.contains("wg") {
                proto = "wireguard"
            } else if pt.contains("openvpn") || pt.contains("ovpn") {
                proto = "openvpn"
            }

            let configDict: [String: Any] = [
                "protocol": proto,
                "peerAddr": server.peerAddr,
                "directMode": !useMaxTurn,
                "maxCallToken": maxCallToken,
                "maxLoginToken": maxLoginToken,
                "maxDeviceID": maxDeviceId,
                "socks5Port": 10808,
                "relayPort": 10900,
                "hysteriaURI": server.hysteriaURI.isEmpty ? server.uri : server.hysteriaURI,
                "privateKey": server.privateKey,
                "publicKey": server.publicKey,
                "presharedKey": server.presharedKey,
                "clientIP": server.clientIP,
                "dns": server.dns,
                "mtu": server.mtu,
                "jc": server.jc,
                "jmin": server.jmin,
                "jmax": server.jmax,
                "s1": server.s1,
                "s2": server.s2,
                "h1": server.h1,
                "h2": server.h2,
                "h3": server.h3,
                "h4": server.h4,
                "openvpnConfig": server.openvpnConfig,
                "username": server.username,
                "password": server.password
            ]

            if let jsonData = try? JSONSerialization.data(withJSONObject: configDict),
               let jsonString = String(data: jsonData, encoding: .utf8) {
                vpn.start(configJSON: jsonString)
            }
        }
    }

    private var serverListSheet: some View {
        NavigationView {
            List {
                ForEach(serverManager.servers) { s in
                    HStack(spacing: 12) {
                        ProtocolBadge(proto: s.protocolType)

                        VStack(alignment: .leading, spacing: 4) {
                            Text(s.name)
                                .font(.headline)
                                .lineLimit(1)
                            Text(s.peerAddr.isEmpty ? s.uri : s.peerAddr)
                                .font(.caption)
                                .foregroundColor(.secondary)
                                .lineLimit(1)
                        }

                        Spacer()

                        Button(action: {
                            UIPasteboard.general.string = s.toShareLink()
                            let gen = UINotificationFeedbackGenerator()
                            gen.notificationOccurred(.success)
                        }) {
                            Image(systemName: "square.and.arrow.up")
                                .foregroundColor(.blue)
                        }
                        .buttonStyle(.borderless)

                        if s.id == serverManager.selectedServerId {
                            Image(systemName: "checkmark")
                                .foregroundColor(.blue)
                                .font(.headline)
                        }
                    }
                    .contentShape(Rectangle())
                    .onTapGesture {
                        serverManager.selectedServerId = s.id
                        showServerList = false
                    }
                }
                .onDelete(perform: serverManager.removeServer)
            }
            .navigationTitle("Servers")
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button(action: {
                        showServerList = false
                        showGeneratorSheet = true
                    }) {
                        Label("New Key", systemImage: "key.fill")
                    }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { showServerList = false }
                }
            }
        }
    }

    private var importSheet: some View {
        NavigationView {
            VStack(spacing: 16) {
                Text("Paste any 1-line key (awg://, wg://, ovpn://, hy2://) or raw WireGuard / OpenVPN config file:")
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .frame(maxWidth: .infinity, alignment: .leading)

                TextEditor(text: $importText)
                    .padding(8)
                    .background(Color(UIColor.secondarySystemGroupedBackground))
                    .cornerRadius(8)
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(Color.secondary.opacity(0.3), lineWidth: 1)
                    )
                    .frame(height: 200)

                Button("Import from Clipboard") {
                    if let clip = UIPasteboard.general.string {
                        importText = clip
                    }
                }
                .buttonStyle(.bordered)

                Spacer()
            }
            .padding()
            .navigationTitle("Add Node / Key")
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("Cancel") { showImportSheet = false }
                }
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Add") {
                        serverManager.importFromClipboardOrText(importText)
                        importText = ""
                        showImportSheet = false
                    }
                    .disabled(importText.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
    }

    private var settingsSheet: some View {
        NavigationView {
            Form {
                Section(header: Text("MAX Credentials")) {
                    SecureField("Call Token", text: $maxCallToken)
                    TextField("Device ID", text: $maxDeviceId)
                    SecureField("Login Token (Optional)", text: $maxLoginToken)
                }

                Section(header: Text("About")) {
                    HStack {
                        Text("Version")
                        Spacer()
                        Text("1.0.0 (TrollStore)")
                            .foregroundColor(.secondary)
                    }
                    HStack {
                        Text("Protocols")
                        Spacer()
                        Text("AWG, WG, HY2, OVPN UDP")
                            .foregroundColor(.secondary)
                    }
                    HStack {
                        Text("MAX-TURN Bypass")
                        Spacer()
                        Text("VK / OK Datagrams")
                            .foregroundColor(.secondary)
                    }
                }
            }
            .navigationTitle("Settings")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { showSettingsSheet = false }
                }
            }
        }
    }
}

// MARK: - Protocol Badge
struct ProtocolBadge: View {
    let proto: String

    var body: some View {
        let (label, color): (String, Color) = {
            let p = proto.lowercased()
            if p.contains("amnezia") || p.contains("awg") { return ("AWG", .purple) }
            if p.contains("wireguard") || p.contains("wg") { return ("WG", .blue) }
            if p.contains("hysteria") || p.contains("hy2") { return ("HY2", .orange) }
            if p.contains("openvpn") || p.contains("ovpn") { return ("OVPN", .teal) }
            return ("VPN", .secondary)
        }()

        Text(label)
            .font(.system(size: 10, weight: .bold))
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(color.opacity(0.15))
            .foregroundColor(color)
            .cornerRadius(4)
    }
}

// MARK: - Key Generator Sheet
struct KeyGeneratorSheet: View {
    @ObservedObject var serverManager: ServerManager
    @Environment(\.dismiss) private var dismiss

    @State private var protocolChoice: Int = 0 // 0: AmneziaWG, 1: WireGuard
    @State private var serverEndpoint: String = "185.195.23.44:51820"
    @State private var serverPubKey: String = "c2VydmVyLXB1YmtleS0zMndnZXhhbXBsZTEyMzQ1Ng=="
    @State private var clientIP: String = "10.8.0.2"
    @State private var serverName: String = "Amnezia MAX Node"

    // Obfuscation parameters
    @State private var jc: Int = 4
    @State private var jmin: Int = 40
    @State private var jmax: Int = 70
    @State private var s1: Int = 15
    @State private var s2: Int = 20
    @State private var h1: Int = 1
    @State private var h2: Int = 2
    @State private var h3: Int = 3
    @State private var h4: Int = 4

    @State private var generatedPrivateKey: String = ""
    @State private var showCopiedAlert: Bool = false

    var body: some View {
        NavigationView {
            Form {
                Section(header: Text("Protocol")) {
                    Picker("Type", selection: $protocolChoice) {
                        Text("AmneziaWG (DPI Bypass)").tag(0)
                        Text("Standard WireGuard").tag(1)
                    }
                    .pickerStyle(.segmented)
                }

                Section(header: Text("Server Details")) {
                    TextField("Server Name", text: $serverName)
                    TextField("Endpoint (IP:Port)", text: $serverEndpoint)
                    TextField("Server Public Key", text: $serverPubKey)
                    TextField("Client IP", text: $clientIP)
                }

                if protocolChoice == 0 {
                    Section(header: Text("AmneziaWG Obfuscation (Anti-DPI)")) {
                        HStack {
                            Text("Jc: \(jc) | Jmin: \(jmin) | Jmax: \(jmax)")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        HStack {
                            Text("S1: \(s1) | S2: \(s2)")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }
                        HStack {
                            Text("H1..H4: \(h1), \(h2), \(h3), \(h4)")
                                .font(.caption)
                                .foregroundColor(.secondary)
                        }

                        Button(action: randomizeObfuscation) {
                            Label("Randomize DPI Obfuscation", systemImage: "dice.fill")
                        }
                    }
                }

                Section(header: Text("1-Line Share Link")) {
                    Text(currentLink)
                        .font(.system(size: 11, design: .monospaced))
                        .lineLimit(4)
                        .padding(.vertical, 4)

                    Button(action: {
                        UIPasteboard.general.string = currentLink
                        let gen = UINotificationFeedbackGenerator()
                        gen.notificationOccurred(.success)
                        showCopiedAlert = true
                    }) {
                        Label("Copy Link for Telegram", systemImage: "doc.on.doc")
                    }

                    Button(action: saveToServerList) {
                        Label("Save to Server List", systemImage: "plus.circle.fill")
                            .foregroundColor(.green)
                    }
                }
            }
            .navigationTitle("Key Generator")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { dismiss() }
                }
            }
            .onAppear {
                generateNewKeyPair()
            }
            .alert("Copied!", isPresented: $showCopiedAlert) {
                Button("OK", role: .cancel) { }
            } message: {
                Text("1-line key copied to clipboard. You can paste and share it in Telegram or Happ.")
            }
        }
    }

    private var currentLink: String {
        let isAwg = (protocolChoice == 0)
        let proto = isAwg ? "awg" : "wg"
        var link = "\(proto)://\(generatedPrivateKey)@\(serverEndpoint)?ip=\(clientIP)&pub=\(serverPubKey)&dns=1.1.1.1"
        if isAwg {
            link += "&jc=\(jc)&jmin=\(jmin)&jmax=\(jmax)&s1=\(s1)&s2=\(s2)&h1=\(h1)&h2=\(h2)&h3=\(h3)&h4=\(h4)"
        }
        let encodedName = serverName.addingPercentEncoding(withAllowedCharacters: .urlFragmentAllowed) ?? serverName
        return link + "#" + encodedName
    }

    private func generateNewKeyPair() {
        var privBytes = [UInt8](repeating: 0, count: 32)
        _ = SecRandomCopyBytes(kSecRandomDefault, 32, &privBytes)
        privBytes[0] &= 248
        privBytes[31] = (privBytes[31] & 127) | 64
        generatedPrivateKey = Data(privBytes).base64EncodedString()
    }

    private func randomizeObfuscation() {
        jc = Int.random(in: 3...8)
        jmin = Int.random(in: 35...55)
        jmax = Int.random(in: 65...110)
        s1 = Int.random(in: 15...40)
        s2 = Int.random(in: 20...50)
        h1 = Int.random(in: 100000...999999999)
        h2 = Int.random(in: 100000...999999999)
        h3 = Int.random(in: 100000...999999999)
        h4 = Int.random(in: 100000...999999999)
    }

    private func saveToServerList() {
        let isAwg = (protocolChoice == 0)
        let s = ProxyServer(
            name: serverName,
            protocolType: isAwg ? "AmneziaWG" : "WireGuard",
            peerAddr: serverEndpoint,
            uri: currentLink,
            privateKey: generatedPrivateKey,
            publicKey: serverPubKey,
            clientIP: clientIP,
            dns: "1.1.1.1",
            mtu: 1420,
            jc: isAwg ? jc : 0,
            jmin: isAwg ? jmin : 0,
            jmax: isAwg ? jmax : 0,
            s1: isAwg ? s1 : 0,
            s2: isAwg ? s2 : 0,
            h1: isAwg ? h1 : 0,
            h2: isAwg ? h2 : 0,
            h3: isAwg ? h3 : 0,
            h4: isAwg ? h4 : 0
        )
        serverManager.addServer(s)
        dismiss()
    }
}
