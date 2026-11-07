import SwiftUI

struct ContentView: View {
    @StateObject private var vpn = VPNController()
    @StateObject private var serverManager = ServerManager()

    @AppStorage("useMaxTurn") private var useMaxTurn: Bool = true
    @AppStorage("maxCallToken") private var maxCallToken: String = "$laipx1vh6JVC3FdLMaGIhnRqikbMifOXTGwUUo7MNsVbuyy1grABxPinzsb0DNn8Vw36hdYaO3"
    @AppStorage("maxDeviceId") private var maxDeviceId: String = "68470aa6-34a4-49ad-bf2f-2a534624270b"
    @AppStorage("maxLoginToken") private var maxLoginToken: String = ""

    @State private var showServerList = false
    @State private var showImportSheet = false
    @State private var showSettingsSheet = false
    @State private var importText = ""

    var body: some View {
        NavigationView {
            ZStack {
                Color(UIColor.systemGroupedBackground)
                    .ignoresSafeArea()

                VStack(spacing: 24) {
                    // Status Badge
                    HStack {
                        Circle()
                            .fill(vpn.isConnected ? Color.green : (vpn.isConnecting ? Color.orange : Color.gray))
                            .frame(width: 10, height: 10)
                        Text(vpn.status)
                            .font(.system(size: 15, weight: .medium))
                            .foregroundColor(.secondary)
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
                                VStack(alignment: .leading, spacing: 2) {
                                    Text("MAX-TURN Relay")
                                        .font(.system(size: 16, weight: .semibold))
                                    Text("Bypasses DPI blocks via VK/MAX TURN")
                                        .font(.system(size: 12))
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
                        HStack {
                            Image(systemName: "server.rack")
                                .font(.system(size: 20))
                                .foregroundColor(.blue)

                            VStack(alignment: .leading, spacing: 4) {
                                Text(serverManager.selectedServer?.name ?? "Select Server")
                                    .font(.system(size: 16, weight: .semibold))
                                    .foregroundColor(.primary)
                                    .lineLimit(1)

                                Text(serverManager.selectedServer?.protocolType ?? "No node selected")
                                    .font(.system(size: 12))
                                    .foregroundColor(.secondary)
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
                ToolbarItem(placement: .navigationBarTrailing) {
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

            let configDict: [String: Any] = [
                "hysteriaURI": server.uri,
                "maxCallToken": maxCallToken,
                "maxLoginToken": maxLoginToken,
                "maxDeviceID": maxDeviceId,
                "socks5Port": 10808,
                "relayPort": 10900,
                "directMode": !useMaxTurn
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
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(s.name).font(.headline)
                            Text(s.protocolType).font(.caption).foregroundColor(.secondary)
                        }
                        Spacer()
                        if s.id == serverManager.selectedServerId {
                            Image(systemName: "checkmark")
                                .foregroundColor(.blue)
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
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") { showServerList = false }
                }
            }
        }
    }

    private var importSheet: some View {
        NavigationView {
            VStack(spacing: 16) {
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
            .navigationTitle("Add Node / Subscription")
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

