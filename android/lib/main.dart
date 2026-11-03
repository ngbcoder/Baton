import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'pairing_page.dart';
import 'onboarding_page.dart';

void main() => runApp(const BatonApp());

class BatonApp extends StatefulWidget {
  const BatonApp({super.key});

  @override
  State<BatonApp> createState() => _BatonAppState();
}

class _BatonAppState extends State<BatonApp> {
  bool _initialized = false;
  bool _showOnboarding = true;

  @override
  void initState() {
    super.initState();
    _checkFirstRun();
  }

  Future<void> _checkFirstRun() async {
    final prefs = await SharedPreferences.getInstance();
    final completed = prefs.getBool('onboarding_completed') ?? false;
    setState(() {
      _showOnboarding = !completed;
      _initialized = true;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Baton',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple, brightness: Brightness.dark),
        useMaterial3: true,
      ),
      home: !_initialized
          ? const Scaffold(body: Center(child: CircularProgressIndicator()))
          : _showOnboarding
              ? OnboardingPage(
                  onFinished: () => setState(() => _showOnboarding = false),
                )
              : const HomePage(),
    );
  }
}

class HomePage extends StatefulWidget {
  const HomePage({super.key});

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  static const _channel = MethodChannel('baton');
  String _status = 'Ready';
  List<Map<dynamic, dynamic>> _computers = [];
  List<Map<dynamic, dynamic>> _headphones = [];
  bool _loading = false;

  @override
  void initState() {
    super.initState();
    _loadData();
  }

  Future<void> _loadData() async {
    setState(() => _loading = true);
    try {
      final comps = await _channel.invokeMethod<List<dynamic>>('getPairedComputers');
      final phones = await _channel.invokeMethod<List<dynamic>>('getHeadphones');
      setState(() {
        _computers = comps?.cast<Map<dynamic, dynamic>>() ?? [];
        _headphones = phones?.cast<Map<dynamic, dynamic>>() ?? [];
        _loading = false;
      });
    } on PlatformException catch (e) {
      setState(() {
        _status = 'Error loading data: ${e.message}';
        _loading = false;
      });
    }
  }

  Future<void> _sendHandoff(String computerName) async {
    try {
      await _channel.invokeMethod('sendHandoff');
      if (!mounted) return;
      setState(() => _status = 'Handoff requested to $computerName');
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Sent headphones to $computerName')),
      );
    } on PlatformException catch (e) {
      setState(() => _status = 'Handoff failed: ${e.message}');
    }
  }

  Future<void> _pullHandoff(String computerName) async {
    try {
      await _channel.invokeMethod('pullHandoff');
      if (!mounted) return;
      setState(() => _status = 'Pulled headphones from $computerName');
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Pulled headphones from $computerName')),
      );
    } on PlatformException catch (e) {
      setState(() => _status = 'Pull failed: ${e.message}');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Baton Audio Handoff'),
        actions: [
          IconButton(
            icon: const Icon(Icons.help_outline),
            tooltip: 'Onboarding & Guide',
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(
                builder: (_) => OnboardingPage(
                  onFinished: () => Navigator.of(context).pop(),
                ),
              ),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.settings),
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => const SettingsPage()),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.qr_code_scanner),
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(builder: (_) => const PairingPage()),
            ),
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : RefreshIndicator(
              onRefresh: _loadData,
              child: ListView(
                padding: const EdgeInsets.all(16),
                children: [
                  Card(
                    child: Padding(
                      padding: const EdgeInsets.all(16),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('Status', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
                          const SizedBox(height: 8),
                          Text(_status, style: const TextStyle(color: Colors.greenAccent)),
                        ],
                      ),
                    ),
                  ),
                  const SizedBox(height: 16),
                  const Text('Paired Computers', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  ..._computers.map((comp) => Card(
                        child: ListTile(
                          leading: const Icon(Icons.computer),
                          title: Text(comp['name'] ?? 'Computer'),
                          subtitle: Text(comp['address'] ?? ''),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              FilledButton.tonal(
                                onPressed: () => _sendHandoff(comp['name']),
                                child: const Text('Send'),
                              ),
                              const SizedBox(width: 8),
                              OutlinedButton(
                                onPressed: () => _pullHandoff(comp['name']),
                                child: const Text('Pull'),
                              ),
                            ],
                          ),
                        ),
                      )),
                  const SizedBox(height: 16),
                  const Text('Headphones', style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold)),
                  const SizedBox(height: 8),
                  if (_headphones.isEmpty)
                    const Card(
                      child: ListTile(
                        leading: Icon(Icons.bluetooth_searching),
                        title: Text('No Paired Bluetooth Headphones Found'),
                        subtitle: Text('Pair headphones in Android Settings or grant Bluetooth permission'),
                      ),
                    ),
                  ..._headphones.map((phone) => Card(
                        child: ListTile(
                          leading: const Icon(Icons.headphones),
                          title: Text(phone['name'] ?? 'Headphones'),
                          subtitle: Text(phone['address'] ?? ''),
                          trailing: Chip(
                            label: Text(phone['connected'] == true ? 'Connected' : 'Available'),
                            backgroundColor: phone['connected'] == true ? Colors.green.shade900 : Colors.grey.shade800,
                          ),
                        ),
                      )),
                ],
              ),
            ),
    );
  }
}

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key});

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  String _disconnectStrategy = 'Multipoint';
  bool _fgKeepalive = true;
  bool _autoCallHandoff = true;
  String _cooldown = '30s';
  final TextEditingController _deviceNameController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _loadSettings();
  }

  Future<void> _loadSettings() async {
    final prefs = await SharedPreferences.getInstance();
    setState(() {
      _disconnectStrategy = prefs.getString('disconnect_strategy') ?? 'Multipoint';
      _fgKeepalive = prefs.getBool('fg_keepalive') ?? true;
      _autoCallHandoff = prefs.getBool('auto_call_handoff') ?? true;
      _cooldown = prefs.getString('cooldown_timer') ?? '30s';
      _deviceNameController.text = prefs.getString('device_name') ?? 'My Android Phone';
    });
  }

  Future<void> _saveSetting(String key, dynamic value) async {
    final prefs = await SharedPreferences.getInstance();
    if (value is String) {
      await prefs.setString(key, value);
    } else if (value is bool) {
      await prefs.setBool(key, value);
    }
  }

  @override
  void dispose() {
    _deviceNameController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Baton Settings')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          TextField(
            controller: _deviceNameController,
            decoration: const InputDecoration(
              labelText: 'Device Name',
              border: OutlineInputBorder(),
            ),
            onChanged: (val) => _saveSetting('device_name', val),
          ),
          const SizedBox(height: 16),
          ListTile(
            title: const Text('Disconnect Strategy'),
            subtitle: const Text('Multipoint / ADB fallback'),
            trailing: DropdownButton<String>(
              value: _disconnectStrategy,
              items: const [
                DropdownMenuItem(value: 'Multipoint', child: Text('Multipoint')),
                DropdownMenuItem(value: 'ADB', child: Text('ADB / Shizuku')),
                DropdownMenuItem(value: 'Fallback', child: Text('Pause & Let Connect')),
              ],
              onChanged: (val) {
                if (val != null) {
                  setState(() => _disconnectStrategy = val);
                  _saveSetting('disconnect_strategy', val);
                }
              },
            ),
          ),
          SwitchListTile(
            title: const Text('Foreground Service Keepalive'),
            subtitle: const Text('Keep TCP connection active in background'),
            value: _fgKeepalive,
            onChanged: (val) {
              setState(() => _fgKeepalive = val);
              _saveSetting('fg_keepalive', val);
            },
          ),
          SwitchListTile(
            title: const Text('Automatic Incoming Call Handoff'),
            subtitle: const Text('Automatically pull headphones to phone on call'),
            value: _autoCallHandoff,
            onChanged: (val) {
              setState(() => _autoCallHandoff = val);
              _saveSetting('auto_call_handoff', val);
            },
          ),
          ListTile(
            title: const Text('Cooldown Timer'),
            subtitle: const Text('Seconds between auto triggers'),
            trailing: DropdownButton<String>(
              value: _cooldown,
              items: const [
                DropdownMenuItem(value: '15s', child: Text('15s')),
                DropdownMenuItem(value: '30s', child: Text('30s')),
                DropdownMenuItem(value: '60s', child: Text('60s')),
              ],
              onChanged: (val) {
                if (val != null) {
                  setState(() => _cooldown = val);
                  _saveSetting('cooldown_timer', val);
                }
              },
            ),
          ),
        ],
      ),
    );
  }
}
