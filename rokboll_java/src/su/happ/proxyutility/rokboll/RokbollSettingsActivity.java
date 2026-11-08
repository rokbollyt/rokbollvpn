package su.happ.proxyutility.rokboll;

import android.app.AlertDialog;
import android.app.ProgressDialog;
import android.content.Context;
import android.content.DialogInterface;
import android.content.SharedPreferences;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.text.InputType;
import android.view.View;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.Toast;

import androidx.appcompat.app.AppCompatActivity;
import androidx.appcompat.widget.Toolbar;
import su.happ.proxyutility.ui.foundation.component.HappForwardField;

import org.json.JSONObject;

public class RokbollSettingsActivity extends AppCompatActivity implements View.OnClickListener {
    final Handler mHandler = new Handler(Looper.getMainLooper());

    private HappForwardField mMaxAccountField;
    private HappForwardField mMaxLinkField;
    private HappForwardField mWrapUdpField;
    private HappForwardField mVkLinkField;

    private int getId(String name) {
        return getResources().getIdentifier(name, "id", getPackageName());
    }

    private int getLayoutId(String name) {
        return getResources().getIdentifier(name, "layout", getPackageName());
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        int layoutId = getLayoutId("activity_rokboll_settings");
        if (layoutId != 0) {
            setContentView(layoutId);
        }

        View toolbarView = findViewById(getId("toolbar"));
        if (toolbarView instanceof Toolbar) {
            Toolbar toolbar = (Toolbar) toolbarView;
            toolbar.setNavigationOnClickListener(this);
        }

        View fMaxAccount = findViewById(getId("forward_max_account"));
        if (fMaxAccount instanceof HappForwardField) {
            mMaxAccountField = (HappForwardField) fMaxAccount;
            mMaxAccountField.setOnClickListener(this);
        }

        View fMaxLink = findViewById(getId("forward_max_link"));
        if (fMaxLink instanceof HappForwardField) {
            mMaxLinkField = (HappForwardField) fMaxLink;
            mMaxLinkField.setOnClickListener(this);
        }

        View fWrapUdp = findViewById(getId("forward_wrap_udp"));
        if (fWrapUdp instanceof HappForwardField) {
            mWrapUdpField = (HappForwardField) fWrapUdp;
            mWrapUdpField.setOnClickListener(this);
        }

        View fVkLink = findViewById(getId("forward_vk_link"));
        if (fVkLink instanceof HappForwardField) {
            mVkLinkField = (HappForwardField) fVkLink;
            mVkLinkField.setOnClickListener(this);
        }

        refreshFields();
    }

    @Override
    public void onClick(View v) {
        int id = v.getId();
        if (id == getId("toolbar")) {
            finish();
        } else if (id == getId("forward_max_account")) {
            showMaxPhoneDialog();
        } else if (id == getId("forward_max_link")) {
            showEditLinkDialog("Ссылка звонка MAX по умолчанию", RokbollBridge.KEY_MAX_LINK, RokbollBridge.DEFAULT_MAX_LINK);
        } else if (id == getId("forward_wrap_udp")) {
            SharedPreferences p = RokbollBridge.getPrefs(this);
            boolean current = p.getBoolean(RokbollBridge.KEY_WRAP_UDP_MAXTURN, true);
            p.edit().putBoolean(RokbollBridge.KEY_WRAP_UDP_MAXTURN, !current).apply();
            refreshFields();
            Toast.makeText(this, !current ? "MAX TURN поверх UDP: ВКЛЮЧЕНО" : "MAX TURN поверх UDP: ВЫКЛЮЧЕНО", Toast.LENGTH_SHORT).show();
        } else if (id == getId("forward_vk_link")) {
            showEditLinkDialog("Ссылка звонка VK по умолчанию", RokbollBridge.KEY_VK_LINK, "");
        }
    }

    void refreshFields() {
        SharedPreferences p = RokbollBridge.getPrefs(this);

        if (mMaxAccountField != null) {
            String phone = p.getString(RokbollBridge.KEY_PHONE, null);
            String logTok = p.getString(RokbollBridge.KEY_LOGIN_TOKEN, null);
            if (phone != null && !phone.isEmpty() && logTok != null && !logTok.isEmpty()) {
                mMaxAccountField.setValue(phone);
                mMaxAccountField.setDescription("Сессия активна");
            } else {
                mMaxAccountField.setValue("Не выполнен");
                mMaxAccountField.setDescription("Нажмите для входа по SMS");
            }
        }

        if (mMaxLinkField != null) {
            String link = p.getString(RokbollBridge.KEY_MAX_LINK, RokbollBridge.DEFAULT_MAX_LINK);
            if (link.length() > 20) {
                link = link.substring(0, 18) + "...";
            }
            mMaxLinkField.setValue(link);
        }

        if (mWrapUdpField != null) {
            boolean wrap = p.getBoolean(RokbollBridge.KEY_WRAP_UDP_MAXTURN, true);
            if (wrap) {
                mWrapUdpField.setValue("Включено");
                mWrapUdpField.setDescription("Любой UDP ключ идет через MAX TURN relay");
            } else {
                mWrapUdpField.setValue("Выключено");
                mWrapUdpField.setDescription("Прямое подключение к серверу ключа");
            }
        }

        if (mVkLinkField != null) {
            String link = p.getString(RokbollBridge.KEY_VK_LINK, "");
            if (link.isEmpty()) {
                mVkLinkField.setValue("Не задана");
            } else if (link.length() > 20) {
                mVkLinkField.setValue(link.substring(0, 18) + "...");
            } else {
                mVkLinkField.setValue(link);
            }
        }
    }

    private void showEditLinkDialog(final String title, final String prefKey, final String defaultVal) {
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle(title);

        final EditText input = new EditText(this);
        SharedPreferences p = RokbollBridge.getPrefs(this);
        input.setText(p.getString(prefKey, defaultVal));
        input.setSingleLine(true);

        LinearLayout layout = new LinearLayout(this);
        layout.setOrientation(LinearLayout.VERTICAL);
        int pad = (int) (16 * getResources().getDisplayMetrics().density);
        layout.setPadding(pad, pad, pad, pad);
        layout.addView(input);
        builder.setView(layout);

        builder.setPositiveButton("Сохранить", new EditDialogListener(this, prefKey, input));
        builder.setNegativeButton("Отмена", null);
        builder.show();
    }

    static class EditDialogListener implements DialogInterface.OnClickListener {
        private final RokbollSettingsActivity mActivity;
        private final String mPrefKey;
        private final EditText mInput;

        EditDialogListener(RokbollSettingsActivity activity, String prefKey, EditText input) {
            this.mActivity = activity;
            this.mPrefKey = prefKey;
            this.mInput = input;
        }

        @Override
        public void onClick(DialogInterface dialog, int which) {
            String val = mInput.getText().toString().trim();
            RokbollBridge.getPrefs(mActivity).edit().putString(mPrefKey, val).apply();
            mActivity.refreshFields();
            Toast.makeText(mActivity, "Сохранено", Toast.LENGTH_SHORT).show();
        }
    }

    private void showMaxPhoneDialog() {
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("Вход в MAX");

        final EditText input = new EditText(this);
        input.setHint("+79001234567");
        input.setInputType(InputType.TYPE_CLASS_PHONE);
        SharedPreferences p = RokbollBridge.getPrefs(this);
        input.setText(p.getString(RokbollBridge.KEY_PHONE, ""));

        LinearLayout layout = new LinearLayout(this);
        layout.setOrientation(LinearLayout.VERTICAL);
        int pad = (int) (16 * getResources().getDisplayMetrics().density);
        layout.setPadding(pad, pad, pad, pad);
        layout.addView(input);
        builder.setView(layout);

        builder.setPositiveButton("Получить код", new PhoneDialogListener(this, input));
        builder.setNegativeButton("Отмена", null);
        builder.show();
    }

    static class PhoneDialogListener implements DialogInterface.OnClickListener {
        private final RokbollSettingsActivity mActivity;
        private final EditText mInput;

        PhoneDialogListener(RokbollSettingsActivity activity, EditText input) {
            this.mActivity = activity;
            this.mInput = input;
        }

        @Override
        public void onClick(DialogInterface dialog, int which) {
            String phone = mInput.getText().toString().trim();
            if (phone.isEmpty()) {
                Toast.makeText(mActivity, "Введите номер телефона", Toast.LENGTH_SHORT).show();
                return;
            }
            mActivity.requestCodeAsync(phone);
        }
    }

    void requestCodeAsync(final String phone) {
        final ProgressDialog pd = new ProgressDialog(this);
        pd.setMessage("Запрос SMS кода...");
        pd.setCancelable(false);
        pd.show();

        new Thread(new RequestCodeTask(this, phone, pd)).start();
    }

    static class RequestCodeTask implements Runnable {
        private final RokbollSettingsActivity mActivity;
        private final String mPhone;
        private final ProgressDialog mPd;

        RequestCodeTask(RokbollSettingsActivity activity, String phone, ProgressDialog pd) {
            this.mActivity = activity;
            this.mPhone = phone;
            this.mPd = pd;
        }

        @Override
        public void run() {
            try {
                final JSONObject res = RokbollBridge.requestCode(mActivity, mPhone);
                mActivity.mHandler.post(new RequestCodeResultTask(mActivity, mPd, mPhone, res));
            } catch (final Throwable t) {
                mActivity.mHandler.post(new ErrorToastTask(mActivity, mPd, "Ошибка запроса: " + t.getMessage()));
            }
        }
    }

    static class RequestCodeResultTask implements Runnable {
        private final RokbollSettingsActivity mActivity;
        private final ProgressDialog mPd;
        private final String mPhone;
        private final JSONObject mRes;

        RequestCodeResultTask(RokbollSettingsActivity activity, ProgressDialog pd, String phone, JSONObject res) {
            this.mActivity = activity;
            this.mPd = pd;
            this.mPhone = phone;
            this.mRes = res;
        }

        @Override
        public void run() {
            mPd.dismiss();
            if (mRes.has("error") && !mRes.isNull("error")) {
                String err = mRes.optString("error", "Неизвестная ошибка");
                Toast.makeText(mActivity, "Ошибка: " + err, Toast.LENGTH_LONG).show();
            } else {
                String captchaUrl = mRes.optString("captcha_url", "");
                String trackToken = mRes.optString("track_token", "");
                if (!captchaUrl.isEmpty() && trackToken.isEmpty()) {
                    mActivity.showCaptchaDialog(mPhone, captchaUrl);
                } else {
                    mActivity.showMaxVerifyDialog(mPhone, trackToken);
                }
            }
        }
    }

    void showCaptchaDialog(final String phone, final String captchaUrl) {
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("Подтверждение (Я не робот)");

        final android.webkit.WebView webView = new android.webkit.WebView(this);
        android.webkit.WebSettings ws = webView.getSettings();
        ws.setJavaScriptEnabled(true);
        ws.setDomStorageEnabled(true);
        ws.setDatabaseEnabled(true);
        ws.setAllowFileAccess(true);
        ws.setAllowContentAccess(true);
        String ua = ws.getUserAgentString();
        if (ua != null) {
            ws.setUserAgentString(ua.replace("; wv", ""));
        }
        if (android.os.Build.VERSION.SDK_INT >= 21) {
            ws.setMixedContentMode(android.webkit.WebSettings.MIXED_CONTENT_ALWAYS_ALLOW);
        }

        final String INJECT_JS = "javascript:(function() {" +
            "if (window.__injected) return;" +
            "window.__injected = true;" +
            "window.addEventListener('message', function(e) {" +
            "  try {" +
            "    if (!e || !e.data) return;" +
            "    var d = e.data;" +
            "    var t = (typeof d === 'string') ? d : (d.token || (d.params && d.params.token) || (d.response && d.response.success_token) || d.success_token);" +
            "    if (t && typeof t === 'string' && t.length > 20) { window.AppBridge.onToken(t); }" +
            "  } catch(err) {}" +
            "});" +
            "var _o = XMLHttpRequest.prototype.open;" +
            "var _s = XMLHttpRequest.prototype.send;" +
            "XMLHttpRequest.prototype.open = function(m, u) { this._u = u; return _o.apply(this, arguments); };" +
            "XMLHttpRequest.prototype.send = function() {" +
            "  this.addEventListener('load', function() {" +
            "    try {" +
            "      if (this.responseText && this.responseText.indexOf('success_token') !== -1) {" +
            "        var p = JSON.parse(this.responseText);" +
            "        var tok = (p.response && p.response.success_token) || p.success_token;" +
            "        if (tok) window.AppBridge.onToken(tok);" +
            "      }" +
            "    } catch(err) {}" +
            "  });" +
            "  return _s.apply(this, arguments);" +
            "};" +
            "if (window.fetch) {" +
            "  var _f = window.fetch;" +
            "  window.fetch = function() {" +
            "    return _f.apply(this, arguments).then(function(res) {" +
            "      try {" +
            "        res.clone().text().then(function(txt) {" +
            "          if (txt && txt.indexOf('success_token') !== -1) {" +
            "            var p = JSON.parse(txt);" +
            "            var tok = (p.response && p.response.success_token) || p.success_token;" +
            "            if (tok) window.AppBridge.onToken(tok);" +
            "          }" +
            "        });" +
            "      } catch(err) {}" +
            "      return res;" +
            "    });" +
            "  };" +
            "}" +
        "})()";

        final AlertDialog dialog;

        class AppBridge {
            private boolean handled = false;

            @android.webkit.JavascriptInterface
            public void onToken(final String token) {
                if (handled || token == null || token.isEmpty()) return;
                handled = true;
                mHandler.post(new Runnable() {
                    @Override
                    public void run() {
                        try {
                            // Dialog will be dismissed when requestCodeWithCaptchaAsync starts
                        } catch (Exception ignored) {}
                        requestCodeWithCaptchaAsync(phone, token);
                    }
                });
            }
        }

        final AppBridge bridge = new AppBridge();
        webView.addJavascriptInterface(bridge, "AppBridge");

        webView.setWebViewClient(new android.webkit.WebViewClient() {
            @Override
            public void onPageStarted(android.webkit.WebView view, String url, android.graphics.Bitmap favicon) {
                view.loadUrl(INJECT_JS);
            }

            @Override
            public void onPageFinished(android.webkit.WebView view, String url) {
                view.loadUrl(INJECT_JS);
            }

            @Override
            public void onReceivedSslError(android.webkit.WebView view, android.webkit.SslErrorHandler handler, android.net.http.SslError error) {
                handler.proceed();
            }
        });

        webView.setWebChromeClient(new android.webkit.WebChromeClient());

        LinearLayout layout = new LinearLayout(this);
        layout.setOrientation(LinearLayout.VERTICAL);
        int h = (int) (380 * getResources().getDisplayMetrics().density);
        layout.addView(webView, new LinearLayout.LayoutParams(
            LinearLayout.LayoutParams.MATCH_PARENT,
            h
        ));

        builder.setView(layout);
        builder.setNegativeButton("Отмена", null);
        dialog = builder.create();

        // Register dismissal callback in bridge
        webView.addJavascriptInterface(new Object() {
            @android.webkit.JavascriptInterface
            public void onToken(final String token) {
                mHandler.post(new Runnable() {
                    @Override
                    public void run() {
                        try { dialog.dismiss(); } catch (Exception ignored) {}
                    }
                });
                bridge.onToken(token);
            }
        }, "AppBridge");

        webView.loadUrl(captchaUrl);
        dialog.show();
    }

    void requestCodeWithCaptchaAsync(final String phone, final String captchaToken) {
        final ProgressDialog pd = new ProgressDialog(this);
        pd.setMessage("Отправка SMS кода...");
        pd.setCancelable(false);
        pd.show();

        new Thread(new Runnable() {
            @Override
            public void run() {
                try {
                    final JSONObject res = RokbollBridge.requestCode(RokbollSettingsActivity.this, phone, captchaToken);
                    mHandler.post(new RequestCodeResultTask(RokbollSettingsActivity.this, pd, phone, res));
                } catch (final Throwable t) {
                    mHandler.post(new ErrorToastTask(RokbollSettingsActivity.this, pd, "Ошибка запроса SMS: " + t.getMessage()));
                }
            }
        }).start();
    }

    void showMaxVerifyDialog(final String phone, final String trackToken) {
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("Код из SMS");

        final EditText input = new EditText(this);
        input.setHint("123456");
        input.setInputType(InputType.TYPE_CLASS_NUMBER);

        LinearLayout layout = new LinearLayout(this);
        layout.setOrientation(LinearLayout.VERTICAL);
        int pad = (int) (16 * getResources().getDisplayMetrics().density);
        layout.setPadding(pad, pad, pad, pad);
        layout.addView(input);
        builder.setView(layout);

        builder.setPositiveButton("Подтвердить", new VerifyDialogListener(this, phone, trackToken, input));
        builder.setNegativeButton("Отмена", null);
        builder.show();
    }

    static class VerifyDialogListener implements DialogInterface.OnClickListener {
        private final RokbollSettingsActivity mActivity;
        private final String mPhone;
        private final String mTrackToken;
        private final EditText mInput;

        VerifyDialogListener(RokbollSettingsActivity activity, String phone, String trackToken, EditText input) {
            this.mActivity = activity;
            this.mPhone = phone;
            this.mTrackToken = trackToken;
            this.mInput = input;
        }

        @Override
        public void onClick(DialogInterface dialog, int which) {
            String code = mInput.getText().toString().trim();
            if (code.isEmpty()) {
                Toast.makeText(mActivity, "Введите код", Toast.LENGTH_SHORT).show();
                return;
            }
            mActivity.verifyCodeAsync(mPhone, mTrackToken, code);
        }
    }

    void verifyCodeAsync(final String phone, final String trackToken, final String code) {
        final ProgressDialog pd = new ProgressDialog(this);
        pd.setMessage("Проверка кода...");
        pd.setCancelable(false);
        pd.show();

        new Thread(new VerifyCodeTask(this, phone, trackToken, code, pd)).start();
    }

    static class VerifyCodeTask implements Runnable {
        private final RokbollSettingsActivity mActivity;
        private final String mPhone;
        private final String mTrackToken;
        private final String mCode;
        private final ProgressDialog mPd;

        VerifyCodeTask(RokbollSettingsActivity activity, String phone, String trackToken, String code, ProgressDialog pd) {
            this.mActivity = activity;
            this.mPhone = phone;
            this.mTrackToken = trackToken;
            this.mCode = code;
            this.mPd = pd;
        }

        @Override
        public void run() {
            try {
                final JSONObject res = RokbollBridge.verifyCode(mActivity, mTrackToken, mCode, mPhone);
                mActivity.mHandler.post(new VerifyCodeResultTask(mActivity, mPd, mPhone, res));
            } catch (final Throwable t) {
                mActivity.mHandler.post(new ErrorToastTask(mActivity, mPd, "Ошибка проверки: " + t.getMessage()));
            }
        }
    }

    static class VerifyCodeResultTask implements Runnable {
        private final RokbollSettingsActivity mActivity;
        private final ProgressDialog mPd;
        private final String mPhone;
        private final JSONObject mRes;

        VerifyCodeResultTask(RokbollSettingsActivity activity, ProgressDialog pd, String phone, JSONObject res) {
            this.mActivity = activity;
            this.mPd = pd;
            this.mPhone = phone;
            this.mRes = res;
        }

        @Override
        public void run() {
            mPd.dismiss();
            String err = mRes.optString("error", "");
            if (err.startsWith("PASSWORD_CHALLENGE:")) {
                String[] parts = err.split(":");
                String trackId = parts.length > 1 ? parts[1] : "";
                String hint = parts.length > 2 ? parts[2] : "";
                mActivity.showMaxPasswordDialog(mPhone, trackId, hint);
                return;
            }
            if (mRes.optBoolean("password_required", false) || mRes.optBoolean("need_password", false)) {
                String trackId = mRes.optString("track_id", "");
                String hint = mRes.optString("hint", "");
                mActivity.showMaxPasswordDialog(mPhone, trackId, hint);
            } else if (mRes.has("error") && !mRes.isNull("error")) {
                Toast.makeText(mActivity, "Ошибка: " + err, Toast.LENGTH_LONG).show();
            } else {
                Toast.makeText(mActivity, "Вход в MAX успешно выполнен!", Toast.LENGTH_LONG).show();
                mActivity.refreshFields();
            }
        }
    }

    void showMaxPasswordDialog(final String phone, final String trackId, final String hint) {
        AlertDialog.Builder builder = new AlertDialog.Builder(this);
        builder.setTitle("Облачный пароль 2FA");
        if (hint != null && !hint.isEmpty()) {
            builder.setMessage("Подсказка: " + hint);
        }

        final EditText input = new EditText(this);
        input.setHint("Введите пароль");
        input.setInputType(InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_PASSWORD);

        LinearLayout layout = new LinearLayout(this);
        layout.setOrientation(LinearLayout.VERTICAL);
        int pad = (int) (16 * getResources().getDisplayMetrics().density);
        layout.setPadding(pad, pad, pad, pad);
        layout.addView(input);
        builder.setView(layout);

        builder.setPositiveButton("Подтвердить", new DialogInterface.OnClickListener() {
            @Override
            public void onClick(DialogInterface dialog, int which) {
                String pwd = input.getText().toString();
                if (pwd.isEmpty()) {
                    Toast.makeText(RokbollSettingsActivity.this, "Введите пароль", Toast.LENGTH_SHORT).show();
                    return;
                }
                verifyPasswordAsync(phone, trackId, pwd);
            }
        });
        builder.setNegativeButton("Отмена", null);
        builder.show();
    }

    void verifyPasswordAsync(final String phone, final String trackId, final String password) {
        final ProgressDialog pd = new ProgressDialog(this);
        pd.setMessage("Проверка пароля...");
        pd.setCancelable(false);
        pd.show();

        new Thread(new Runnable() {
            @Override
            public void run() {
                try {
                    final JSONObject res = RokbollBridge.verifyPassword(RokbollSettingsActivity.this, trackId, password, phone);
                    mHandler.post(new Runnable() {
                        @Override
                        public void run() {
                            pd.dismiss();
                            if (res.has("error") && !res.isNull("error")) {
                                String err = res.optString("error", "Неверный пароль");
                                Toast.makeText(RokbollSettingsActivity.this, "Ошибка: " + err, Toast.LENGTH_LONG).show();
                            } else {
                                Toast.makeText(RokbollSettingsActivity.this, "Вход в MAX успешно выполнен!", Toast.LENGTH_LONG).show();
                                refreshFields();
                            }
                        }
                    });
                } catch (final Throwable t) {
                    mHandler.post(new ErrorToastTask(RokbollSettingsActivity.this, pd, "Ошибка проверки: " + t.getMessage()));
                }
            }
        }).start();
    }

    static class ErrorToastTask implements Runnable {
        private final RokbollSettingsActivity mActivity;
        private final ProgressDialog mPd;
        private final String mMsg;

        ErrorToastTask(RokbollSettingsActivity activity, ProgressDialog pd, String msg) {
            this.mActivity = activity;
            this.mPd = pd;
            this.mMsg = msg;
        }

        @Override
        public void run() {
            mPd.dismiss();
            Toast.makeText(mActivity, mMsg, Toast.LENGTH_LONG).show();
        }
    }
}
