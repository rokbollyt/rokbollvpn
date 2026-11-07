import Foundation

struct ProxyServer: Identifiable, Codable, Equatable {
    var id: String = UUID().uuidString
    var name: String
    var uri: String
    var protocolType: String
    var pingMs: Int?
}

@MainActor
final class ServerManager: ObservableObject {
    @Published var servers: [ProxyServer] = []
    @Published var selectedServerId: String = ""

    private let storageKey = "HappPort_Servers"
    private let selectedKey = "HappPort_SelectedServer"

    init() {
        loadServers()
        if servers.isEmpty {
            // Default Japan server from session
            let defaultUri = "hysteria2://02df03e3-3029-4c95-bffe-8f211b480672@edge.hankli1214.xyz:443/?sni=hank.hankli1214.xyz&obfs=salamander&obfs-password=MeJA9qgoYMH4UMunHowisrbd&upmbps=50&downmbps=500#Japan-MAX"
            addServer(from: defaultUri, name: "Japan (MAX Fast)")
        }
    }

    var selectedServer: ProxyServer? {
        servers.first { $0.id == selectedServerId } ?? servers.first
    }

    func addServer(from uri: String, name: String? = nil) {
        let trimmed = uri.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return }

        let parsedName = name ?? extractName(from: trimmed)
        let proto = extractProtocol(from: trimmed)

        let server = ProxyServer(
            name: parsedName,
            uri: trimmed,
            protocolType: proto
        )

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

    func importFromClipboardOrText(_ text: String) {
        let lines = text.components(separatedBy: .newlines)
        for line in lines {
            let item = line.trimmingCharacters(in: .whitespacesAndNewlines)
            if item.hasPrefix("hysteria2://") || item.hasPrefix("hy2://") ||
               item.hasPrefix("vless://") || item.hasPrefix("vmess://") {
                addServer(from: item)
            }
        }
    }

    private func extractName(from uri: String) -> String {
        if let fragment = uri.components(separatedBy: "#").last, !fragment.isEmpty, fragment != uri {
            return fragment.removingPercentEncoding ?? fragment
        }
        if let host = URL(string: uri)?.host {
            return host
        }
        return "Proxy Server"
    }

    private func extractProtocol(from uri: String) -> String {
        if uri.hasPrefix("hysteria2://") || uri.hasPrefix("hy2://") {
            return "Hysteria 2"
        } else if uri.hasPrefix("vless://") {
            return "VLESS"
        } else if uri.hasPrefix("vmess://") {
            return "VMess"
        } else if uri.hasPrefix("ss://") {
            return "Shadowsocks"
        }
        return "Unknown"
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

