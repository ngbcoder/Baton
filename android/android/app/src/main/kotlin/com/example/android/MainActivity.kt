package com.example.android

import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val channelName = "baton"

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName).setMethodCallHandler { call, result ->
            when (call.method) {
                "getPairedComputers" -> result.success(emptyList<Map<String, Any>>())
                "getHeadphones" -> result.success(emptyList<Map<String, Any>>())
                "sendHandoff" -> result.success(true)
                else -> result.notImplemented()
            }
        }
    }
}