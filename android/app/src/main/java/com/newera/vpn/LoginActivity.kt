package com.newera.vpn

import android.annotation.SuppressLint
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.os.Bundle
import android.webkit.CookieManager
import android.webkit.WebResourceRequest
import android.webkit.WebView
import android.webkit.WebViewClient
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder

@SuppressLint("SetJavaScriptEnabled")
class LoginActivity : AppCompatActivity() {

    private val redirectUri = "https://vpn.newera2625.top/oauth/app-callback"

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val web = WebView(this)
        web.settings.javaScriptEnabled = true
        web.settings.domStorageEnabled = true
        setContentView(web)

        web.webViewClient = object : WebViewClient() {
            override fun shouldOverrideUrlLoading(view: WebView?, request: WebResourceRequest?): Boolean {
                val url = request?.url?.toString() ?: return false
                if (url.startsWith(redirectUri)) {
                    handleCallback(url)
                    return true
                }
                return false
            }

            override fun onPageStarted(view: WebView?, url: String?, favicon: Bitmap?) {
                super.onPageStarted(view, url, favicon)
                if (url != null && url.startsWith(redirectUri)) {
                    handleCallback(url)
                }
            }
        }

        val authUrl = "https://sso.newera2625.top/oauth/authorize" +
            "?client_id=vpn-site" +
            "&redirect_uri=" + URLEncoder.encode(redirectUri, "UTF-8") +
            "&response_type=code&state=android"
        web.loadUrl(authUrl)
    }

    private fun handleCallback(url: String) {
        val code = url.substringAfter("code=", "").substringBefore("&")
        if (code.isEmpty()) {
            Toast.makeText(this, "登录失败：缺少授权码", Toast.LENGTH_SHORT).show()
            finish()
            return
        }
        // 换 token
        Thread {
            try {
                val tokenUrl = "https://sso.newera2625.top/oauth/token" +
                    "?grant_type=authorization_code" +
                    "&code=" + URLEncoder.encode(code, "UTF-8") +
                    "&client_id=vpn-site" +
                    "&redirect_uri=" + URLEncoder.encode(redirectUri, "UTF-8")
                val conn = URL(tokenUrl).openConnection() as HttpURLConnection
                conn.connectTimeout = 10000
                conn.readTimeout = 10000
                val body = conn.inputStream.bufferedReader().readText()
                val d = JSONObject(body)
                val token = d.optString("access_token", "")
                if (token.isEmpty()) {
                    runOnUiThread {
                        Toast.makeText(this, "登录失败：${d.optString("error", "未知错误")}", Toast.LENGTH_SHORT).show()
                        finish()
                    }
                    return@Thread
                }
                // 拉用户信息
                val u = getUserInfo(token)
                val sp = getSharedPreferences("vpn", Context.MODE_PRIVATE)
                sp.edit()
                    .putString("token", token)
                    .putString("nickname", u?.optString("nickname", "") ?: "")
                    .putString("avatar", u?.optString("avatar", "") ?: "")
                    .apply()
                runOnUiThread {
                    Toast.makeText(this, "登录成功", Toast.LENGTH_SHORT).show()
                    startActivity(Intent(this, MainActivity::class.java).apply { flags = Intent.FLAG_ACTIVITY_CLEAR_TOP })
                    finish()
                }
            } catch (e: Exception) {
                runOnUiThread {
                    Toast.makeText(this, "登录失败: ${e.message}", Toast.LENGTH_SHORT).show()
                    finish()
                }
            }
        }.start()
    }

    private fun getUserInfo(token: String): JSONObject? {
        return try {
            val conn = URL("https://sso.newera2625.top/api/status").openConnection() as HttpURLConnection
            conn.setRequestProperty("Cookie", "auth_token=$token")
            conn.connectTimeout = 10000
            val body = conn.inputStream.bufferedReader().readText()
            val d = JSONObject(body)
            if (d.getInt("code") == 200) d.getJSONObject("data").optJSONObject("user") else null
        } catch (e: Exception) {
            null
        }
    }
}
