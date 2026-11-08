package su.happ.proxyutility.rokboll;

import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.net.Uri;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;
import android.view.View;
import android.widget.Toast;

import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.File;
import java.io.FileOutputStream;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.lang.reflect.Field;
import java.lang.reflect.Method;
import java.net.URLDecoder;
import java.net.URLEncoder;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public final class RokbollBridge {
    private static final String TAG = "RokbollBridge";
    public static final String PREFS_NAME = "rokboll_prefs";

    public static final String KEY_PHONE = "max_phone";
    public static final String KEY_LOGIN_TOKEN = "max_login_token";
    public static final String KEY_CALL_TOKEN = "max_call_token";
    public static final String KEY_MAX_LINK = "max_call_link";
    public static final String KEY_VK_LINK = "vk_call_link";
    public static final String KEY_DEVICE_ID = "device_id";
    public static final String KEY_WRAP_UDP_MAXTURN = "wrap_udp_maxturn";
    public static final int LOCAL_RELAY_PORT = 11590;

    public static final String DEFAULT_MAX_LINK = "flBY3rVQonTIxS_G81y8qMRQCbhquUQgFi4kLGDtTNQ";

    private static Process sTunnelProcess = null;
    private static String sCurrentKind = null;

    public static SharedPreferences getPrefs(Context context) {
        return context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE);
    }

    public static String getDeviceId(Context context) {
        SharedPreferences p = getPrefs(context);
        String dev = p.getString(KEY_DEVICE_ID, null);
        if (dev == null || dev.trim().isEmpty()) {
            dev = UUID.randomUUID().toString();
            p.edit().putString(KEY_DEVICE_ID, dev).apply();
        }
        return dev;
    }

    public static File getBinaryFile(Context context) {
        try {
            String nativeDir = context.getApplicationInfo().nativeLibraryDir;
            File direct = new File(nativeDir, "librokboll.so");
            if (direct.exists() && direct.canExecute()) {
                return direct;
            }
            if (direct.exists()) {
                direct.setExecutable(true, false);
                if (direct.canExecute()) {
                    return direct;
                }
            }

            File target = new File(context.getFilesDir(), "librokboll.bin");
            if (target.exists() && target.length() > 1000000 && target.canExecute()) {
                return target;
            }

            if (direct.exists()) {
                copyFile(direct, target);
                target.setExecutable(true, false);
                return target;
            }
        } catch (Throwable t) {
            Log.e(TAG, "getBinaryFile error: " + t.getMessage(), t);
        }
        return null;
    }

    private static void copyFile(File src, File dst) throws Exception {
        InputStream in = new java.io.FileInputStream(src);
        try {
            FileOutputStream out = new FileOutputStream(dst);
            try {
                byte[] buf = new byte[65536];
                int read;
                while ((read = in.read(buf)) > 0) {
                    out.write(buf, 0, read);
                }
            } finally {
                out.close();
            }
        } finally {
            in.close();
        }
    }

    public static synchronized void stopTunnel() {
        if (sTunnelProcess != null) {
            try {
                Log.i(TAG, "Stopping active tunnel process...");
                sTunnelProcess.destroy();
            } catch (Throwable t) {
                Log.w(TAG, "Error destroying tunnel: " + t.getMessage());
            }
            sTunnelProcess = null;
            sCurrentKind = null;
        }
    }

    public static synchronized boolean isTunnelRunning() {
        return sTunnelProcess != null;
    }

    public static synchronized void startTunnelWithArgs(Context context, List<String> extraArgs, String kind) {
        stopTunnel();
        try {
            File bin = getBinaryFile(context);
            if (bin == null || !bin.exists()) {
                Log.e(TAG, "Cannot start tunnel: binary not found");
                return;
            }

            List<String> cmd = new ArrayList<String>();
            cmd.add(bin.getAbsolutePath());
            cmd.add("run");
            cmd.add("-kind");
            cmd.add(kind);
            cmd.add("-listen");
            cmd.add("127.0.0.1:10808");
            cmd.addAll(extraArgs);

            Log.i(TAG, "Launching custom tunnel: " + cmd.toString());
            ProcessBuilder pb = new ProcessBuilder(cmd);
            pb.redirectErrorStream(true);
            Process proc = pb.start();

            final Process procRef = proc;
            new Thread(new Runnable() {
                @Override
                public void run() {
                    try {
                        BufferedReader r = new BufferedReader(new InputStreamReader(procRef.getInputStream()));
                        String line;
                        while ((line = r.readLine()) != null) {
                            Log.d(TAG, "[rokboll-tunnel] " + line);
                        }
                    } catch (Throwable ignored) {}
                }
            }, "RokbollLogThread").start();

            sTunnelProcess = proc;
            sCurrentKind = kind;
            Log.i(TAG, "Tunnel process running for kind: " + kind);
        } catch (Throwable t) {
            Log.e(TAG, "startTunnelWithArgs failed: " + t.getMessage(), t);
        }
    }

    public static String executeCli(Context context, String... args) throws Exception {
        File bin = getBinaryFile(context);
        if (bin == null || !bin.exists()) {
            throw new IllegalStateException("Engine binary not found");
        }

        List<String> cmd = new ArrayList<String>();
        cmd.add(bin.getAbsolutePath());
        cmd.addAll(Arrays.asList(args));

        ProcessBuilder pb = new ProcessBuilder(cmd);
        pb.redirectErrorStream(true);
        Process p = pb.start();

        BufferedReader reader = new BufferedReader(new InputStreamReader(p.getInputStream()));
        StringBuilder sb = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) {
            sb.append(line).append("\n");
        }
        p.waitFor();
        return sb.toString().trim();
    }

    public static JSONObject requestCode(Context context, String phone) throws Exception {
        return requestCode(context, phone, "");
    }

    public static JSONObject requestCode(Context context, String phone, String captchaToken) throws Exception {
        String devId = getDeviceId(context);
        if (captchaToken != null && !captchaToken.isEmpty()) {
            String out = executeCli(context, "max-request-code", "-phone", phone, "-captcha", captchaToken, "-device-id", devId);
            return new JSONObject(out);
        }
        String out = executeCli(context, "max-request-code", "-phone", phone, "-device-id", devId);
        return new JSONObject(out);
    }

    public static JSONObject verifyCode(Context context, String trackToken, String code, String phone) throws Exception {
        String devId = getDeviceId(context);
        String out = executeCli(context, "max-verify-code", "-track", trackToken, "-code", code, "-device-id", devId);
        JSONObject obj = new JSONObject(out);
        if (obj.has("login_token") && !obj.isNull("login_token")) {
            String loginTok = obj.getString("login_token");
            String callTok = obj.optString("call_token", "");
            getPrefs(context).edit()
                    .putString(KEY_PHONE, phone)
                    .putString(KEY_LOGIN_TOKEN, loginTok)
                    .putString(KEY_CALL_TOKEN, callTok)
                    .apply();
        }
        return obj;
    }

    public static JSONObject verifyPassword(Context context, String trackId, String password, String phone) throws Exception {
        String devId = getDeviceId(context);
        String out = executeCli(context, "max-verify-password", "-track", trackId, "-password", password, "-device-id", devId);
        JSONObject obj = new JSONObject(out);
        if (obj.has("login_token") && !obj.isNull("login_token")) {
            String loginTok = obj.getString("login_token");
            String callTok = obj.optString("call_token", "");
            getPrefs(context).edit()
                    .putString(KEY_PHONE, phone)
                    .putString(KEY_LOGIN_TOKEN, loginTok)
                    .putString(KEY_CALL_TOKEN, callTok)
                    .apply();
        }
        return obj;
    }

    /**
     * Intercepts import of custom URI keys (minewire://, max-turn://, vk-turn://, awg://, amnezia://, wg://, wireguard://, and INI text)
     * Returns true if handled, false otherwise.
     */
    public static boolean handleImportUri(final Context context, String rawText) {
        if (rawText == null || rawText.trim().isEmpty()) {
            return false;
        }

        String trimmed = rawText.trim();
        int importedCount = 0;
        String lastRemarks = null;

        // Check if raw text is multi-line INI config ([Interface] + [Peer])
        if (trimmed.contains("[Interface]") && trimmed.contains("[Peer]")) {
            try {
                boolean ok = parseAndSaveWireGuard(context, trimmed, true);
                if (ok) {
                    importedCount++;
                    lastRemarks = (trimmed.contains("Jc") || trimmed.contains("H1")) ? "AmneziaWG-Node" : "WireGuard-Node";
                }
            } catch (Throwable t) {
                Log.e(TAG, "Error importing INI config: " + t.getMessage(), t);
            }
        } else {
            String[] lines = trimmed.split("\\r?\\n");
            for (String line : lines) {
                String uriStr = line.trim();
                if (uriStr.startsWith("awg://") || uriStr.startsWith("amnezia://") || uriStr.startsWith("wg://") || uriStr.startsWith("wireguard://")) {
                    try {
                        boolean ok = parseAndSaveWireGuard(context, uriStr, false);
                        if (ok) {
                            importedCount++;
                            String rem = extractRemarks(uriStr);
                            lastRemarks = (rem != null && !rem.isEmpty()) ? rem : (uriStr.startsWith("awg://") || uriStr.startsWith("amnezia://") ? "AmneziaWG-Node" : "WireGuard-Node");
                        }
                    } catch (Throwable t) {
                        Log.e(TAG, "Error importing WG/AWG URI: " + uriStr, t);
                    }
                } else if (uriStr.startsWith("minewire://") || uriStr.startsWith("max-turn://") || uriStr.startsWith("maxturn://") || uriStr.startsWith("vk-turn://") || uriStr.startsWith("vkturn://")) {
                    try {
                        String remarks = extractRemarks(uriStr);
                        if (remarks == null || remarks.isEmpty()) {
                            if (uriStr.startsWith("minewire://")) remarks = "Minewire";
                            else if (uriStr.startsWith("max-turn://") || uriStr.startsWith("maxturn://")) remarks = "MAX-TURN";
                            else remarks = "VK-TURN";
                        }

                        boolean ok = saveCustomServer(context, remarks, uriStr);
                        if (ok) {
                            importedCount++;
                            lastRemarks = remarks;
                        }
                    } catch (Throwable t) {
                        Log.e(TAG, "Error importing URI: " + uriStr, t);
                    }
                }
            }
        }

        if (importedCount > 0) {
            final String msg = importedCount == 1 ? ("Сервер " + lastRemarks + " добавлен!") : ("Импортировано серверов: " + importedCount);
            new Handler(Looper.getMainLooper()).post(new Runnable() {
                @Override
                public void run() {
                    Toast.makeText(context, msg, Toast.LENGTH_SHORT).show();
                }
            });
            return true;
        }

        return false;
    }

    private static String getIniValue(String sectionText, String key) {
        Pattern p = Pattern.compile("(?im)^\\s*" + Pattern.quote(key) + "\\s*=\\s*(.+)$");
        Matcher m = p.matcher(sectionText);
        if (m.find()) {
            String val = m.group(1).trim();
            int commentIdx = val.indexOf('#');
            if (commentIdx >= 0) val = val.substring(0, commentIdx).trim();
            commentIdx = val.indexOf(';');
            if (commentIdx >= 0) val = val.substring(0, commentIdx).trim();
            return val;
        }
        return null;
    }

    private static String cleanBase64(String val) {
        if (val == null) return null;
        val = val.trim();
        // Restore '+' if URL decoder converted it to space
        return val.replace(' ', '+');
    }

    private static boolean parseAndSaveWireGuard(Context context, String text, boolean isIni) {
        try {
            if (isIni) {
                String privKey = cleanBase64(getIniValue(text, "PrivateKey"));
                String address = getIniValue(text, "Address");
                String dns = getIniValue(text, "DNS");
                String pubKey = cleanBase64(getIniValue(text, "PublicKey"));
                String endpoint = getIniValue(text, "Endpoint");
                String psk = cleanBase64(getIniValue(text, "PresharedKey"));
                String mtuStr = getIniValue(text, "MTU");
                int mtu = 1420;
                if (mtuStr != null) {
                    try { mtu = Integer.parseInt(mtuStr); } catch (Throwable ignored) {}
                }

                if (privKey == null || pubKey == null || endpoint == null) {
                    Log.e(TAG, "INI missing PrivateKey, PublicKey, or Endpoint");
                    return false;
                }

                Map<String, String> awgParams = new HashMap<String, String>();
                String jc = getIniValue(text, "Jc");
                String jmin = getIniValue(text, "Jmin");
                String jmax = getIniValue(text, "Jmax");
                String s1 = getIniValue(text, "S1");
                String s2 = getIniValue(text, "S2");
                String h1 = getIniValue(text, "H1");
                String h2 = getIniValue(text, "H2");
                String h3 = getIniValue(text, "H3");
                String h4 = getIniValue(text, "H4");

                boolean isAwg = (jc != null || h1 != null);
                if (jc != null) awgParams.put("jc", jc); else if (isAwg) awgParams.put("jc", "4");
                if (jmin != null) awgParams.put("jmin", jmin); else if (isAwg) awgParams.put("jmin", "40");
                if (jmax != null) awgParams.put("jmax", jmax); else if (isAwg) awgParams.put("jmax", "70");
                if (s1 != null) awgParams.put("s1", s1); else if (isAwg) awgParams.put("s1", "0");
                if (s2 != null) awgParams.put("s2", s2); else if (isAwg) awgParams.put("s2", "0");
                if (h1 != null) awgParams.put("h1", h1); else if (isAwg) awgParams.put("h1", "1");
                if (h2 != null) awgParams.put("h2", h2); else if (isAwg) awgParams.put("h2", "2");
                if (h3 != null) awgParams.put("h3", h3); else if (isAwg) awgParams.put("h3", "3");
                if (h4 != null) awgParams.put("h4", h4); else if (isAwg) awgParams.put("h4", "4");

                String remarks = isAwg ? "AmneziaWG-Node" : "WireGuard-Node";
                return saveWireGuardServer(context, remarks, privKey, address, pubKey, endpoint, psk, mtu, dns, awgParams);
            } else {
                Uri uri = Uri.parse(text);
                String scheme = uri.getScheme() != null ? uri.getScheme().toLowerCase() : "";
                boolean isAwg = scheme.equals("awg") || scheme.equals("amnezia");

                String privKey = uri.getUserInfo();
                if (privKey == null || privKey.isEmpty()) {
                    privKey = uri.getQueryParameter("priv");
                }
                if (privKey == null || privKey.isEmpty()) {
                    privKey = uri.getQueryParameter("privatekey");
                }
                privKey = cleanBase64(privKey);

                String host = uri.getHost();
                int port = uri.getPort();
                if (port <= 0) port = 443;
                String endpoint = host + ":" + port;

                String pubKey = cleanBase64(uri.getQueryParameter("pub"));
                if (pubKey == null || pubKey.isEmpty()) {
                    pubKey = cleanBase64(uri.getQueryParameter("publickey"));
                }

                String address = uri.getQueryParameter("ip");
                if (address == null || address.isEmpty()) {
                    address = uri.getQueryParameter("address");
                }
                if (address == null || address.isEmpty()) {
                    address = "10.104.2.228/32";
                }

                String dns = uri.getQueryParameter("dns");
                String psk = cleanBase64(uri.getQueryParameter("psk"));
                if (psk == null || psk.isEmpty()) {
                    psk = cleanBase64(uri.getQueryParameter("presharedkey"));
                }

                int mtu = 1420;
                String mtuStr = uri.getQueryParameter("mtu");
                if (mtuStr != null) {
                    try { mtu = Integer.parseInt(mtuStr); } catch (Throwable ignored) {}
                }

                Map<String, String> awgParams = new HashMap<String, String>();
                String jc = uri.getQueryParameter("jc");
                String jmin = uri.getQueryParameter("jmin");
                String jmax = uri.getQueryParameter("jmax");
                String s1 = uri.getQueryParameter("s1");
                String s2 = uri.getQueryParameter("s2");
                String h1 = uri.getQueryParameter("h1");
                String h2 = uri.getQueryParameter("h2");
                String h3 = uri.getQueryParameter("h3");
                String h4 = uri.getQueryParameter("h4");

                if (jc != null || h1 != null) {
                    isAwg = true;
                }
                if (isAwg) {
                    awgParams.put("jc", jc != null ? jc : "4");
                    awgParams.put("jmin", jmin != null ? jmin : "40");
                    awgParams.put("jmax", jmax != null ? jmax : "70");
                    awgParams.put("s1", s1 != null ? s1 : "0");
                    awgParams.put("s2", s2 != null ? s2 : "0");
                    awgParams.put("h1", h1 != null ? h1 : "1");
                    awgParams.put("h2", h2 != null ? h2 : "2");
                    awgParams.put("h3", h3 != null ? h3 : "3");
                    awgParams.put("h4", h4 != null ? h4 : "4");
                }

                String remarks = extractRemarks(text);
                if (remarks == null || remarks.isEmpty()) {
                    remarks = isAwg ? "AmneziaWG-Node" : "WireGuard-Node";
                }

                if (privKey == null || pubKey == null || host == null) {
                    Log.e(TAG, "URI missing privKey, pubKey, or host: " + text);
                    return false;
                }

                return saveWireGuardServer(context, remarks, privKey, address, pubKey, endpoint, psk, mtu, dns, awgParams);
            }
        } catch (Throwable t) {
            Log.e(TAG, "parseAndSaveWireGuard error: " + t.getMessage(), t);
            return false;
        }
    }

    private static boolean saveWireGuardServer(Context context, String remarks, String privKey, String address,
                                               String pubKey, String endpoint, String presharedKey, int mtu,
                                               String dns, Map<String, String> awgParams) {
        try {
            int col = endpoint.lastIndexOf(':');
            if (col <= 0) return false;
            String host = endpoint.substring(0, col).trim();
            int port = Integer.parseInt(endpoint.substring(col + 1).trim());

            String peerKey = host + ":" + port;
            SharedPreferences.Editor edit = getPrefs(context).edit();
            if (awgParams != null && !awgParams.isEmpty()) {
                edit.putBoolean("awg_enabled_" + peerKey, true);
                if (awgParams.containsKey("jc")) edit.putInt("awg_jc_" + peerKey, Integer.parseInt(awgParams.get("jc")));
                if (awgParams.containsKey("jmin")) edit.putInt("awg_jmin_" + peerKey, Integer.parseInt(awgParams.get("jmin")));
                if (awgParams.containsKey("jmax")) edit.putInt("awg_jmax_" + peerKey, Integer.parseInt(awgParams.get("jmax")));
                if (awgParams.containsKey("s1")) edit.putInt("awg_s1_" + peerKey, Integer.parseInt(awgParams.get("s1")));
                if (awgParams.containsKey("s2")) edit.putInt("awg_s2_" + peerKey, Integer.parseInt(awgParams.get("s2")));
                if (awgParams.containsKey("h1")) edit.putLong("awg_h1_" + peerKey, Long.parseLong(awgParams.get("h1")));
                if (awgParams.containsKey("h2")) edit.putLong("awg_h2_" + peerKey, Long.parseLong(awgParams.get("h2")));
                if (awgParams.containsKey("h3")) edit.putLong("awg_h3_" + peerKey, Long.parseLong(awgParams.get("h3")));
                if (awgParams.containsKey("h4")) edit.putLong("awg_h4_" + peerKey, Long.parseLong(awgParams.get("h4")));
                Log.i(TAG, "Saved AWG parameters for " + peerKey);
            }
            edit.apply();

            StringBuilder sb = new StringBuilder();
            sb.append("wireguard://");
            sb.append(URLEncoder.encode(privKey, "UTF-8"));
            sb.append("@");
            sb.append(host);
            sb.append(":");
            sb.append(port);
            sb.append("?publickey=").append(URLEncoder.encode(pubKey, "UTF-8"));

            String finalAddress = address != null ? address.trim() : "10.104.2.228/32";
            if (!finalAddress.contains("/")) {
                finalAddress = finalAddress + "/32";
            }
            sb.append("&address=").append(URLEncoder.encode(finalAddress, "UTF-8"));
            sb.append("&reserved=0,0,0");

            if (presharedKey != null && !presharedKey.isEmpty()) {
                sb.append("&presharedkey=").append(URLEncoder.encode(presharedKey, "UTF-8"));
            }
            if (mtu > 0) {
                sb.append("&mtu=").append(mtu);
            }
            if (remarks != null && !remarks.isEmpty()) {
                sb.append("#").append(URLEncoder.encode(remarks, "UTF-8"));
            }

            String wgUri = sb.toString();
            Log.i(TAG, "Generated canonical WireGuard URI for internal parser: " + wgUri);

            Class<?> s37Cls = Class.forName("defpackage.s37");
            Object s37 = s37Cls.newInstance();
            Method mB = s37Cls.getMethod("b", String.class);
            Object m83Obj = mB.invoke(s37, wgUri);
            if (m83Obj == null) {
                Log.e(TAG, "s37.b returned null for " + wgUri);
                return false;
            }

            Field fA = m83Obj.getClass().getField("a");
            Object serverConfig = fA.get(m83Obj);
            if (serverConfig == null) {
                Log.e(TAG, "m83.a is null");
                return false;
            }

            Class<?> serverConfigCls = Class.forName("su.happ.proxyutility.dto.ServerConfig");
            Class<?> nv4Cls = Class.forName("defpackage.nv4");
            Method methodE = nv4Cls.getMethod("e", String.class, serverConfigCls);
            String guid = (String) methodE.invoke(null, "", serverConfig);
            Log.i(TAG, "Successfully saved WireGuard/AWG server, guid=" + guid + " (" + remarks + ")");
            return true;
        } catch (Throwable t) {
            Log.e(TAG, "saveWireGuardServer failed: " + t.getMessage(), t);
            return false;
        }
    }

    private static String extractRemarks(String uriStr) {
        int hashIdx = uriStr.indexOf('#');
        if (hashIdx >= 0 && hashIdx < uriStr.length() - 1) {
            String raw = uriStr.substring(hashIdx + 1);
            try {
                return URLDecoder.decode(raw, "UTF-8");
            } catch (Throwable ignored) {
                return raw;
            }
        }
        return null;
    }

    private static boolean saveCustomServer(Context context, String remarks, String rawUri) {
        try {
            Class<?> eConfigTypeCls = Class.forName("su.happ.proxyutility.dto.enums.EConfigType");
            Object socksType = Enum.valueOf((Class<Enum>) eConfigTypeCls, "SOCKS");

            Class<?> serverConfigCls = Class.forName("su.happ.proxyutility.dto.ServerConfig");
            Class<?> companionCls = Class.forName("su.happ.proxyutility.dto.ServerConfig$Companion");
            Method methodA = companionCls.getMethod("a", eConfigTypeCls);

            Class<?> nv4Cls = Class.forName("defpackage.nv4");
            Method methodE = nv4Cls.getMethod("e", String.class, serverConfigCls);
            Method setRemarks = serverConfigCls.getMethod("m", String.class);
            Method setAdv = serverConfigCls.getMethod("j", String.class);

            Object cfg = methodA.invoke(null, socksType);
            if (cfg != null) {
                setRemarks.invoke(cfg, remarks);
                setAdv.invoke(cfg, rawUri);
                String guid = (String) methodE.invoke(null, "", cfg);

                getPrefs(context).edit()
                        .putString("uri_" + guid, rawUri)
                        .putString("uri_" + remarks, rawUri)
                        .apply();
                return true;
            }
        } catch (Throwable t) {
            Log.e(TAG, "saveCustomServer failed: " + t.getMessage(), t);
        }
        return false;
    }

    public static synchronized void startRawUDPRelay(Context context, String peerHost, int peerPort, int localPort) {
        stopTunnel();
        try {
            File bin = getBinaryFile(context);
            if (bin == null || !bin.exists()) {
                Log.e(TAG, "Cannot start UDP relay: binary not found");
                return;
            }

            SharedPreferences p = getPrefs(context);
            String link = p.getString(KEY_MAX_LINK, DEFAULT_MAX_LINK);
            String loginTok = p.getString(KEY_LOGIN_TOKEN, "");
            String callTok = p.getString(KEY_CALL_TOKEN, "");
            String devId = getDeviceId(context);

            String resolvedHost = peerHost;
            try {
                java.net.InetAddress addr = java.net.InetAddress.getByName(peerHost);
                if (addr != null && addr.getHostAddress() != null) {
                    resolvedHost = addr.getHostAddress();
                    Log.i(TAG, "DNS pre-resolved peer " + peerHost + " -> " + resolvedHost);
                }
            } catch (Throwable t) {
                Log.w(TAG, "DNS pre-resolve failed for " + peerHost + ": " + t.getMessage());
            }

            List<String> cmd = new ArrayList<String>();
            cmd.add(bin.getAbsolutePath());
            cmd.add("udp");
            cmd.add("-listen");
            cmd.add("127.0.0.1:" + localPort);
            cmd.add("-peer");
            cmd.add(resolvedHost + ":" + peerPort);

            if (link != null && !link.isEmpty()) {
                cmd.add("-link");
                cmd.add(link);
            }
            if (loginTok != null && !loginTok.isEmpty()) {
                cmd.add("-login-token");
                cmd.add(loginTok);
            }
            if (callTok != null && !callTok.isEmpty()) {
                cmd.add("-call-token");
                cmd.add(callTok);
            }
            if (devId != null && !devId.isEmpty()) {
                cmd.add("-device-id");
                cmd.add(devId);
            }

            // Check if AmneziaWG parameters are stored for this peer
            String peerKey1 = peerHost + ":" + peerPort;
            String peerKey2 = resolvedHost + ":" + peerPort;
            boolean isAwg = p.getBoolean("awg_enabled_" + peerKey1, false) || p.getBoolean("awg_enabled_" + peerKey2, false);
            String matchedKey = p.contains("awg_h1_" + peerKey1) ? peerKey1 : (p.contains("awg_h1_" + peerKey2) ? peerKey2 : null);
            if (!isAwg && matchedKey != null) {
                isAwg = true;
            }
            if (matchedKey == null) {
                matchedKey = peerKey1;
            }

            if (isAwg) {
                int jc = p.getInt("awg_jc_" + matchedKey, 4);
                int jmin = p.getInt("awg_jmin_" + matchedKey, 40);
                int jmax = p.getInt("awg_jmax_" + matchedKey, 70);
                int s1 = p.getInt("awg_s1_" + matchedKey, 0);
                int s2 = p.getInt("awg_s2_" + matchedKey, 0);
                long h1 = p.getLong("awg_h1_" + matchedKey, 1L);
                long h2 = p.getLong("awg_h2_" + matchedKey, 2L);
                long h3 = p.getLong("awg_h3_" + matchedKey, 3L);
                long h4 = p.getLong("awg_h4_" + matchedKey, 4L);

                cmd.add("-awg");
                cmd.add("-jc");
                cmd.add(String.valueOf(jc));
                cmd.add("-jmin");
                cmd.add(String.valueOf(jmin));
                cmd.add("-jmax");
                cmd.add(String.valueOf(jmax));
                cmd.add("-s1");
                cmd.add(String.valueOf(s1));
                cmd.add("-s2");
                cmd.add(String.valueOf(s2));
                cmd.add("-h1");
                cmd.add(String.valueOf(h1));
                cmd.add("-h2");
                cmd.add(String.valueOf(h2));
                cmd.add("-h3");
                cmd.add(String.valueOf(h3));
                cmd.add("-h4");
                cmd.add(String.valueOf(h4));
                Log.i(TAG, "AmneziaWG parameters attached: Jc=" + jc + ", Jmin=" + jmin + ", Jmax=" + jmax + ", S1=" + s1 + ", S2=" + s2 + ", H1=" + h1 + ", H2=" + h2 + ", H3=" + h3 + ", H4=" + h4);
            }

            Log.i(TAG, "Launching MAX-TURN UDP Relay: " + cmd.toString());
            ProcessBuilder pb = new ProcessBuilder(cmd);
            pb.redirectErrorStream(true);
            Process proc = pb.start();

            final Process procRef = proc;
            new Thread(new Runnable() {
                @Override
                public void run() {
                    try {
                        BufferedReader r = new BufferedReader(new InputStreamReader(procRef.getInputStream()));
                        String line;
                        while ((line = r.readLine()) != null) {
                            Log.d(TAG, "[max-udp-relay] " + line);
                        }
                    } catch (Throwable ignored) {}
                }
            }, "RokbollUDPRelayLogThread").start();

            sTunnelProcess = proc;
            sCurrentKind = "maxturn-udp";
            Log.i(TAG, "MAX-TURN UDP Relay running on port " + localPort + " -> " + peerHost + ":" + peerPort);
        } catch (Throwable t) {
            Log.e(TAG, "startRawUDPRelay failed: " + t.getMessage(), t);
        }
    }

    public static String transformXrayConfig(Object l59Obj, String configJson) {
        if (configJson == null || configJson.trim().isEmpty()) {
            return configJson;
        }
        Context context = null;
        if (l59Obj != null) {
            try {
                Method m = l59Obj.getClass().getMethod("g");
                Object s = m.invoke(l59Obj);
                if (s instanceof Context) {
                    context = (Context) s;
                }
            } catch (Throwable t) {
                Log.w(TAG, "Failed to get context from l59: " + t.getMessage());
            }
        }
        if (context == null) {
            return configJson;
        }

        SharedPreferences p = getPrefs(context);
        boolean wrapEnabled = p.getBoolean(KEY_WRAP_UDP_MAXTURN, true);
        if (!wrapEnabled) {
            Log.d(TAG, "MAX-TURN UDP wrap is disabled, passing config unchanged");
            return configJson;
        }

        if (isTunnelRunning()) {
            Log.d(TAG, "Tunnel already active (" + sCurrentKind + "), skipping UDP wrap");
            return configJson;
        }

        try {
            JSONObject root = new JSONObject(configJson);
            org.json.JSONArray outbounds = root.optJSONArray("outbounds");
            if (outbounds == null || outbounds.length() == 0) {
                return configJson;
            }

            for (int i = 0; i < outbounds.length(); i++) {
                JSONObject outbound = outbounds.getJSONObject(i);
                String tag = outbound.optString("tag", "");
                if (!"proxy".equalsIgnoreCase(tag) && i != 0) {
                    continue;
                }

                JSONObject settings = outbound.optJSONObject("settings");
                if (settings == null) {
                    continue;
                }

                String targetHost = null;
                int targetPort = 0;

                // 0. Direct in settings (Hysteria, Hysteria 2)
                targetHost = settings.optString("address", "");
                targetPort = settings.optInt("port", 0);
                if (!targetHost.isEmpty() && targetPort > 0) {
                    settings.put("address", "127.0.0.1");
                    settings.put("port", LOCAL_RELAY_PORT);
                }

                // 1. servers list (Shadowsocks, Socks)
                if (targetHost == null || targetHost.isEmpty()) {
                    org.json.JSONArray servers = settings.optJSONArray("servers");
                    if (servers != null && servers.length() > 0) {
                        JSONObject srv = servers.getJSONObject(0);
                        targetHost = srv.optString("address", "");
                        targetPort = srv.optInt("port", 0);
                        if (!targetHost.isEmpty() && targetPort > 0) {
                            srv.put("address", "127.0.0.1");
                            srv.put("port", LOCAL_RELAY_PORT);
                        }
                    }
                }

                // 2. peers list (WireGuard)
                if (targetHost == null || targetHost.isEmpty()) {
                    org.json.JSONArray peers = settings.optJSONArray("peers");
                    if (peers != null && peers.length() > 0) {
                        JSONObject peer = peers.getJSONObject(0);
                        String endpoint = peer.optString("endpoint", "");
                        if (!endpoint.isEmpty()) {
                            int col = endpoint.lastIndexOf(':');
                            if (col > 0) {
                                targetHost = endpoint.substring(0, col);
                                try {
                                    targetPort = Integer.parseInt(endpoint.substring(col + 1));
                                    peer.put("endpoint", "127.0.0.1:" + LOCAL_RELAY_PORT);
                                } catch (Throwable ignored) {}
                            }
                        }
                    }
                }

                // 3. vnext list (VLESS, VMess, Trojan)
                if (targetHost == null || targetHost.isEmpty()) {
                    org.json.JSONArray vnext = settings.optJSONArray("vnext");
                    if (vnext != null && vnext.length() > 0) {
                        JSONObject srv = vnext.getJSONObject(0);
                        targetHost = srv.optString("address", "");
                        targetPort = srv.optInt("port", 0);
                        if (!targetHost.isEmpty() && targetPort > 0) {
                            srv.put("address", "127.0.0.1");
                            srv.put("port", LOCAL_RELAY_PORT);
                        }
                    }
                }

                // 4. Direct in outbound
                if (targetHost == null || targetHost.isEmpty()) {
                    targetHost = outbound.optString("address", "");
                    targetPort = outbound.optInt("port", 0);
                    if (!targetHost.isEmpty() && targetPort > 0) {
                        outbound.put("address", "127.0.0.1");
                        outbound.put("port", LOCAL_RELAY_PORT);
                    }
                }

                if (targetHost != null && !targetHost.isEmpty() && targetPort > 0) {
                    // Ensure TLS SNI is preserved so TLS/QUIC handshake succeeds
                    JSONObject streamSettings = outbound.optJSONObject("streamSettings");
                    if (streamSettings == null) {
                        streamSettings = new JSONObject();
                        outbound.put("streamSettings", streamSettings);
                    }
                    JSONObject tlsSettings = streamSettings.optJSONObject("tlsSettings");
                    if (tlsSettings == null) {
                        tlsSettings = new JSONObject();
                        streamSettings.put("tlsSettings", tlsSettings);
                    }
                    String sni = tlsSettings.optString("serverName", "");
                    if (sni.isEmpty() || "127.0.0.1".equals(sni)) {
                        tlsSettings.put("serverName", targetHost);
                    }
                    JSONObject realitySettings = streamSettings.optJSONObject("realitySettings");
                    if (realitySettings != null) {
                        String realitySni = realitySettings.optString("serverName", "");
                        if (realitySni.isEmpty() || "127.0.0.1".equals(realitySni)) {
                            realitySettings.put("serverName", targetHost);
                        }
                    }

                    Log.i(TAG, "Intercepted outbound [" + tag + "] target " + targetHost + ":" + targetPort + " -> routing through MAX-TURN 127.0.0.1:" + LOCAL_RELAY_PORT);
                    startRawUDPRelay(context, targetHost, targetPort, LOCAL_RELAY_PORT);
                    return root.toString();
                }
            }
        } catch (Throwable t) {
            Log.e(TAG, "Error transforming XRay config: " + t.getMessage(), t);
        }

        return configJson;
    }

    public static void onVpnStart(Context context, Object serverConfig) {
        try {
            if (serverConfig == null) {
                stopTunnel();
                return;
            }

            String adv = null;
            try {
                Method mAdv = serverConfig.getClass().getMethod("b");
                Object r = mAdv.invoke(serverConfig);
                if (r != null) adv = r.toString();
            } catch (Throwable ignored) {}

            String remarks = null;
            try {
                Method mRemarks = serverConfig.getClass().getMethod("getRemarks");
                Object r = mRemarks.invoke(serverConfig);
                if (r != null) remarks = r.toString();
            } catch (Throwable ignored) {}
            if (remarks == null) {
                try {
                    Method mRemarks = serverConfig.getClass().getMethod("h");
                    Object r = mRemarks.invoke(serverConfig);
                    if (r != null) remarks = r.toString();
                } catch (Throwable ignored) {}
            }

            String customUri = null;
            if (adv != null && (adv.startsWith("minewire://") || adv.startsWith("max-turn://") || adv.startsWith("vk-turn://"))) {
                customUri = adv;
            } else if (remarks != null) {
                SharedPreferences p = getPrefs(context);
                customUri = p.getString("uri_" + remarks, null);
            }

            if (customUri == null) {
                // Not a custom protocol, ensure tunnel is stopped
                stopTunnel();
                return;
            }

            Log.i(TAG, "Starting VPN for custom URI: " + customUri);
            Uri uri = Uri.parse(customUri);
            String scheme = uri.getScheme();
            if (scheme == null) {
                stopTunnel();
                return;
            }

            scheme = scheme.toLowerCase();
            List<String> args = new ArrayList<String>();
            String host = uri.getHost();
            int port = uri.getPort();

            if ("minewire".equals(scheme)) {
                if (port <= 0) port = 25565;
                args.add("-server");
                args.add(host + ":" + port);

                String password = uri.getUserInfo();
                if (password == null || password.isEmpty()) {
                    password = uri.getQueryParameter("password");
                }
                if (password == null || password.isEmpty()) {
                    password = "dsadk1os01jds0k2-";
                }
                args.add("-password");
                args.add(password);

                String mode = uri.getQueryParameter("mode");
                if (mode == null || mode.isEmpty()) {
                    mode = "fast";
                }
                args.add("-mode");
                args.add(mode);

                startTunnelWithArgs(context, args, "minewire");
            } else if ("max-turn".equals(scheme) || "maxturn".equals(scheme)) {
                if (port <= 0) port = 56000;
                args.add("-peer");
                args.add(host + ":" + port);

                String link = uri.getQueryParameter("link");
                if (link == null || link.isEmpty()) {
                    link = getPrefs(context).getString(KEY_MAX_LINK, DEFAULT_MAX_LINK);
                }
                if (link != null && !link.isEmpty()) {
                    args.add("-link");
                    args.add(link);
                }

                String callTok = uri.getQueryParameter("call_token");
                if (callTok == null || callTok.isEmpty()) {
                    callTok = getPrefs(context).getString(KEY_CALL_TOKEN, "");
                }
                if (callTok != null && !callTok.isEmpty()) {
                    args.add("-call-token");
                    args.add(callTok);
                }

                String loginTok = uri.getQueryParameter("login_token");
                if (loginTok == null || loginTok.isEmpty()) {
                    loginTok = getPrefs(context).getString(KEY_LOGIN_TOKEN, "");
                }
                if (loginTok != null && !loginTok.isEmpty()) {
                    args.add("-login-token");
                    args.add(loginTok);
                }

                String devId = getDeviceId(context);
                args.add("-device-id");
                args.add(devId);

                String streams = uri.getQueryParameter("streams");
                if (streams != null && !streams.isEmpty()) {
                    args.add("-streams");
                    args.add(streams);
                }

                startTunnelWithArgs(context, args, "maxturn");
            } else if ("vk-turn".equals(scheme) || "vkturn".equals(scheme)) {
                if (port <= 0) port = 56000;
                args.add("-peer");
                args.add(host + ":" + port);

                String link = uri.getQueryParameter("link");
                if (link == null || link.isEmpty()) {
                    link = getPrefs(context).getString(KEY_VK_LINK, "");
                }
                if (link != null && !link.isEmpty()) {
                    args.add("-link");
                    args.add(link);
                }

                String streams = uri.getQueryParameter("streams");
                if (streams != null && !streams.isEmpty()) {
                    args.add("-streams");
                    args.add(streams);
                }

                startTunnelWithArgs(context, args, "vkturn");
            } else {
                stopTunnel();
            }
        } catch (Throwable t) {
            Log.e(TAG, "onVpnStart error: " + t.getMessage(), t);
        }
    }

    public static void onVpnStop(Context context) {
        stopTunnel();
    }

    public static void setupOtherSettings(final View rootView) {
        if (rootView == null) return;
        try {
            int id = rootView.getResources().getIdentifier("forward_rokboll_settings", "id", rootView.getContext().getPackageName());
            if (id != 0) {
                View btn = rootView.findViewById(id);
                if (btn != null) {
                    btn.setOnClickListener(new View.OnClickListener() {
                        @Override
                        public void onClick(View v) {
                            try {
                                Context ctx = v.getContext();
                                Intent intent = new Intent(ctx, RokbollSettingsActivity.class);
                                ctx.startActivity(intent);
                            } catch (Throwable t) {
                                Log.e(TAG, "Error opening RokbollSettingsActivity: " + t.getMessage(), t);
                            }
                        }
                    });
                }
            }
        } catch (Throwable t) {
            Log.e(TAG, "setupOtherSettings error: " + t.getMessage(), t);
        }
    }

    public static void excludeApps(android.net.VpnService.Builder builder, Context context) {
        if (builder == null || context == null) return;
        try {
            builder.addDisallowedApplication(context.getPackageName());
        } catch (Throwable ignored) {}
        try {
            builder.addDisallowedApplication("ru.oneme.app");
        } catch (Throwable ignored) {}
    }
}
