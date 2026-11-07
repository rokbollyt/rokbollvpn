import Foundation
import NetworkExtension
import Combine

@MainActor
final class VPNController: ObservableObject {
    @Published var status: String = "Disconnected"
    @Published var isConnected: Bool = false
    @Published var isConnecting: Bool = false

    private var manager: NETunnelProviderManager?
    private let extensionBundleId = "su.rokboll.happport.tunnel"

    init() {
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(statusChanged),
            name: .NEVPNStatusDidChange,
            object: nil
        )
        Task { await loadManager() }
    }

    private func loadManager() async {
        let managers = (try? await NETunnelProviderManager.loadAllFromPreferences()) ?? []
        manager = managers.first { ($0.protocolConfiguration as? NETunnelProviderProtocol)?.providerBundleIdentifier == extensionBundleId } ?? managers.first
        refreshStatus()
    }

    func start(configJSON: String) {
        Task {
            let m = manager ?? NETunnelProviderManager()
            let proto = NETunnelProviderProtocol()
            proto.providerBundleIdentifier = extensionBundleId
            proto.serverAddress = "Happ Port"
            proto.providerConfiguration = [
                "configJSON": configJSON
            ]
            m.protocolConfiguration = proto
            m.localizedDescription = "Happ VPN"
            m.isEnabled = true

            do {
                try await m.saveToPreferences()
                try await m.loadFromPreferences()
                self.manager = m
                try m.connection.startVPNTunnel()
            } catch {
                self.status = "Error: \(error.localizedDescription)"
                self.isConnected = false
                self.isConnecting = false
            }
        }
    }

    func stop() {
        manager?.connection.stopVPNTunnel()
    }

    @objc private func statusChanged() {
        refreshStatus()
    }

    private func refreshStatus() {
        guard let conn = manager?.connection else {
            status = "Disconnected"
            isConnected = false
            isConnecting = false
            return
        }

        switch conn.status {
        case .connected:
            status = "Connected"
            isConnected = true
            isConnecting = false
        case .connecting, .reasserting:
            status = "Connecting..."
            isConnected = false
            isConnecting = true
        case .disconnecting:
            status = "Disconnecting..."
            isConnected = false
            isConnecting = true
        default:
            status = "Disconnected"
            isConnected = false
            isConnecting = false
        }
    }
}

