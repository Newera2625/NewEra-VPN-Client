package com.newera.vpn

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.Button
import android.widget.TextView
import android.widget.Toast
import androidx.appcompat.app.AlertDialog
import androidx.appcompat.app.AppCompatActivity
import androidx.recyclerview.widget.LinearLayoutManager
import androidx.recyclerview.widget.RecyclerView
import androidx.swiperefreshlayout.widget.SwipeRefreshLayout
import org.json.JSONArray
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL

class MainActivity : AppCompatActivity() {

    private lateinit var refreshLayout: SwipeRefreshLayout
    private lateinit var recycler: RecyclerView
    private lateinit var statusText: TextView
    private lateinit var loginBtn: Button
    private val adapter = NodeAdapter()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        statusText = findViewById(R.id.status_text)
        loginBtn = findViewById(R.id.login_btn)
        refreshLayout = findViewById(R.id.refresh_layout)
        recycler = findViewById(R.id.node_list)
        recycler.layoutManager = LinearLayoutManager(this)
        recycler.adapter = adapter

        refreshLayout.setOnRefreshListener { loadNodes() }
        loginBtn.setOnClickListener {
            startActivity(Intent(this, LoginActivity::class.java))
        }

        renderLoginState()
        loadNodes()
    }

    override fun onResume() {
        super.onResume()
        renderLoginState()
    }

    private fun getToken(): String? {
        val sp = getSharedPreferences("vpn", Context.MODE_PRIVATE)
        return sp.getString("token", null)
    }

    private fun renderLoginState() {
        val token = getToken()
        val sp = getSharedPreferences("vpn", Context.MODE_PRIVATE)
        val nickname = sp.getString("nickname", null)
        if (token != null && nickname != null) {
            statusText.text = nickname
            loginBtn.visibility = View.GONE
        } else {
            statusText.text = "未登录"
            loginBtn.visibility = View.VISIBLE
        }
    }

    private fun loadNodes() {
        refreshLayout.isRefreshing = true
        Thread {
            try {
                val conn = URL("https://vpn.newera2625.top/api/nodes").openConnection() as HttpURLConnection
                conn.connectTimeout = 10000
                conn.readTimeout = 10000
                val body = conn.inputStream.bufferedReader().readText()
                val d = JSONObject(body)
                val nodes = d.getJSONObject("data").getJSONArray("nodes")
                runOnUiThread {
                    adapter.setData(parseNodes(nodes))
                    refreshLayout.isRefreshing = false
                }
            } catch (e: Exception) {
                runOnUiThread {
                    Toast.makeText(this, "节点加载失败: ${e.message}", Toast.LENGTH_SHORT).show()
                    refreshLayout.isRefreshing = false
                }
            }
        }.start()
    }

    private fun parseNodes(arr: JSONArray): List<Node> {
        val list = mutableListOf<Node>()
        for (i in 0 until arr.length()) {
            val o = arr.getJSONObject(i)
            list.add(
                Node(
                    o.getString("id"),
                    o.getString("country"),
                    o.getString("country_code"),
                    o.getString("protocol"),
                    o.getInt("latency")
                )
            )
        }
        return list
    }

    private fun connectNode(node: Node) {
        val token = getToken() ?: run {
            Toast.makeText(this, "请先登录", Toast.LENGTH_SHORT).show()
            return
        }
        Thread {
            try {
                val conn = URL("https://vpn.newera2625.top/api/nodes/${node.id}/proxy").openConnection() as HttpURLConnection
                conn.connectTimeout = 10000
                conn.readTimeout = 10000
                conn.setRequestProperty("Cookie", "auth_token=$token")
                val body = conn.inputStream.bufferedReader().readText()
                val d = JSONObject(body)
                if (d.getInt("code") != 200) {
                    runOnUiThread { Toast.makeText(this, d.optString("message", "连接失败"), Toast.LENGTH_SHORT).show() }
                    return@Thread
                }
                val data = d.getJSONObject("data")
                val addr = "${data.getString("ip")}:${data.getInt("port")}"
                runOnUiThread { showProxyDialog(node, addr) }
            } catch (e: Exception) {
                runOnUiThread { Toast.makeText(this, "获取代理失败: ${e.message}", Toast.LENGTH_SHORT).show() }
            }
        }.start()
    }

    private fun showProxyDialog(node: Node, addr: String) {
        AlertDialog.Builder(this)
            .setTitle("连接节点 · ${node.country}")
            .setMessage(
                "代理地址：$addr\n协议：${node.protocol.uppercase()}\n延迟：${node.latency} ms\n\n" +
                    "使用方法：\n1. 点击下方「复制代理地址」\n2. 打开 系统设置 → WLAN → 长按当前WiFi → 修改网络 → 代理选择「手动」\n3. 粘贴主机名和端口后保存"
            )
            .setPositiveButton("复制代理地址") { _, _ ->
                val cm = getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
                cm.setPrimaryClip(ClipData.newPlainText("proxy", addr))
                Toast.makeText(this, "已复制 $addr", Toast.LENGTH_SHORT).show()
            }
            .setNegativeButton("关闭", null)
            .show()
    }

    data class Node(val id: String, val country: String, val code: String, val protocol: String, val latency: Int)

    inner class NodeAdapter : RecyclerView.Adapter<NodeAdapter.Holder>() {
        private val items = mutableListOf<Node>()

        fun setData(list: List<Node>) {
            items.clear()
            items.addAll(list)
            notifyDataSetChanged()
        }

        override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): Holder {
            val v = LayoutInflater.from(parent.context).inflate(R.layout.item_node, parent, false)
            return Holder(v)
        }

        override fun onBindViewHolder(holder: Holder, position: Int) {
            val n = items[position]
            holder.country.text = n.country
            holder.meta.text = "${n.protocol.uppercase()} · ${n.latency} ms"
            holder.code.text = n.code
            holder.connect.setOnClickListener { connectNode(n) }
        }

        override fun getItemCount() = items.size

        inner class Holder(v: View) : RecyclerView.ViewHolder(v) {
            val code: TextView = v.findViewById(R.id.node_code)
            val country: TextView = v.findViewById(R.id.node_country)
            val meta: TextView = v.findViewById(R.id.node_meta)
            val connect: Button = v.findViewById(R.id.node_connect)
        }
    }
}
