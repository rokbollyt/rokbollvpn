import Foundation
import NetworkExtension

/// PacketTunnelProvider bridges system L3 IP packets from iOS to the Go router stack.
/// It also handles bypass routes to prevent the MAX-TURN relay and DNS queries
/// from looping back into the tunnel.
class PacketTunnelProvider: NEPacketTunnelProvider {

    /// Networks that must bypass the VPN tunnel:
    /// MAX / VK / OK TURN servers, auth endpoints, and direct DNS servers.
    static let bypassRoutes: [NEIPv4Route] = {
        let cidrs: [(String, String)] = [
            // VK / OK / MAX IP ranges
            ("217.20.144.0", "255.255.240.0"),
            ("217.20.160.0", "255.255.224.0"),
            ("95.163.0.0", "255.255.0.0"),
            ("185.16.244.0", "255.255.252.0"),
            ("185.16.248.0", "255.255.252.0"),
            ("128.140.168.0", "255.255.248.0"),
            // Fallback DNS resolvers
            ("77.88.8.8", "255.255.255.255"),
            ("77.88.8.1", "255.255.255.255"),
            ("8.8.8.8", "255.255.255.255"),
            ("1.1.1.1", "255.255.255.255"),
        ]
        return cidrs.map { NEIPv4Route(destinationAddress: $0.0, subnetMask: $0.1) }
    }()

    private var isRunning = false

    override func startTunnel(options: [String: NSObject]?, completionHandler: @escaping (Error?) -> Void) {
        let conf = (protocolConfiguration as? NETunnelProviderProtocol)?.providerConfiguration ?? [:]
        let configJSON = (conf["configJSON"] as? String) ?? "{}"

        let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: "127.0.0.1")
        let ipv4 = NEIPv4Settings(addresses: ["10.254.1.2"], subnetMasks: ["255.255.255.0"])
        ipv4.includedRoutes = [NEIPv4Route.default()]
        ipv4.excludedRoutes = Self.bypassRoutes
        settings.ipv4Settings = ipv4
        settings.mtu = 1500

        let dns = NEDNSSettings(servers: ["198.18.0.1"])
        dns.matchDomains = [""]
        settings.dnsSettings = dns

        setTunnelNetworkSettings(settings) { [weak self] error in
            guard let self = self else { return }
            if let error = error {
                completionHandler(error)
                return
            }

            let rc = configJSON.withCString { ptr in
                RouterStart(UnsafeMutablePointer(mutating: ptr))
            }

            if rc != 0 {
                completionHandler(NSError(domain: "su.rokboll.happport", code: Int(rc),
                    userInfo: [NSLocalizedDescriptionKey: "RouterStart failed code: \(rc)"]))
                return
            }

            self.isRunning = true
            self.startReadLoop()
            self.startWriteLoop()
            completionHandler(nil)
        }
    }

    override func stopTunnel(with reason: NEProviderStopReason, completionHandler: @escaping () -> Void) {
        isRunning = false
        RouterStop()
        completionHandler()
    }

    /// Read packets from iOS device -> send to Go stack
    private func startReadLoop() {
        guard isRunning else { return }
        packetFlow.readPackets { [weak self] packets, _ in
            guard let self = self, self.isRunning else { return }
            for p in packets {
                p.withUnsafeBytes { raw in
                    if let base = raw.bindMemory(to: CChar.self).baseAddress {
                        RouterWritePacket(UnsafeMutablePointer(mutating: base), Int32(p.count))
                    }
                }
            }
            self.startReadLoop()
        }
    }

    /// Read packets from Go stack -> send to iOS device
    private func startWriteLoop() {
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            let maxLen: Int32 = 65536
            let buf = UnsafeMutablePointer<CChar>.allocate(capacity: Int(maxLen))
            defer { buf.deallocate() }

            while let self = self, self.isRunning {
                let n = RouterReadPacket(buf, maxLen)
                if n > 0 {
                    let data = Data(bytes: buf, count: Int(n))
                    self.packetFlow.writePackets([data], withProtocols: [NSNumber(value: AF_INET)])
                } else {
                    usleep(5000) // 5ms sleep if no packets
                }
            }
        }
    }
}

