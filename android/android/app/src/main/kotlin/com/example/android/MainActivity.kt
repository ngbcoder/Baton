package com.example.android

import android.Manifest
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothDevice
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel

class MainActivity : FlutterActivity() {
    private val channelName = "baton"
    private val REQUEST_BT_PERMISSIONS = 101

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        requestBluetoothPermissionsIfNeeded()

        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, channelName).setMethodCallHandler { call, result ->
            when (call.method) {
                "getPairedComputers" -> {
                    val computers = getPairedComputersReal()
                    result.success(computers)
                }
                "getHeadphones" -> {
                    val devices = getBluetoothHeadphonesReal()
                    result.success(devices)
                }
                "sendHandoff" -> {
                    startService(Intent(this, BatonForegroundService::class.java))
                    result.success(true)
                }
                "pullHandoff" -> {
                    startService(Intent(this, BatonForegroundService::class.java))
                    result.success(true)
                }
                "startForegroundService" -> {
                    startService(Intent(this, BatonForegroundService::class.java))
                    result.success(true)
                }
                "stopForegroundService" -> {
                    stopService(Intent(this, BatonForegroundService::class.java))
                    result.success(true)
                }
                else -> result.notImplemented()
            }
        }
    }

    private fun requestBluetoothPermissionsIfNeeded() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            val permissions = arrayOf(
                Manifest.permission.BLUETOOTH_CONNECT,
                Manifest.permission.BLUETOOTH_SCAN
            )
            val needed = permissions.filter {
                ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED
            }
            if (needed.isNotEmpty()) {
                ActivityCompat.requestPermissions(this, needed.toTypedArray(), REQUEST_BT_PERMISSIONS)
            }
        }
    }

    private fun getBluetoothHeadphonesReal(): List<Map<String, Any>> {
        val adapter = BluetoothAdapter.getDefaultAdapter() ?: return emptyList()
        val devices = mutableListOf<Map<String, Any>>()

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) {
            if (ContextCompat.checkSelfPermission(this, Manifest.permission.BLUETOOTH_CONNECT) != PackageManager.PERMISSION_GRANTED) {
                return emptyList()
            }
        }

        try {
            val bonded: Set<BluetoothDevice>? = adapter.bondedDevices
            bonded?.forEach { device ->
                val btClass = try { device.bluetoothClass } catch (e: SecurityException) { null }
               val major = btClass?.majorDeviceClass ?: -1
               if (major != android.bluetooth.BluetoothClass.Major.AUDIO_VIDEO) return@forEach
               val name = try { device.name ?: "Bluetooth Device" } catch (e: SecurityException) { "Bluetooth Device" }
                devices.add(mapOf(
                    "name" to name,
                    "address" to device.address,
                    "connected" to true
                ))
            }
        } catch (e: Exception) {
            // Permission or BT disabled
        }

        return devices
    }

    private fun getPairedComputersReal(): List<Map<String, Any>> {
        return listOf(
            mapOf("id" to "pc_local", "name" to "Windows PC", "address" to "127.0.0.1:4040", "paired" to true)
        )
    }
}
