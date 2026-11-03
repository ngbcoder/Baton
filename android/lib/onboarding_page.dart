import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'pairing_page.dart';

class OnboardingPage extends StatefulWidget {
  const OnboardingPage({super.key, required this.onFinished});

  final VoidCallback onFinished;

  @override
  State<OnboardingPage> createState() => _OnboardingPageState();
}

class _OnboardingPageState extends State<OnboardingPage> {
  static const _channel = MethodChannel('baton');
  final PageController _pageController = PageController();
  int _currentPage = 0;
  late TextEditingController _nameController;
  bool _permissionsGranted = false;

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: 'My Android Phone');
    _loadSavedName();
  }

  Future<void> _loadSavedName() async {
    final prefs = await SharedPreferences.getInstance();
    final savedName = prefs.getString('device_name');
    if (savedName != null && savedName.isNotEmpty) {
      _nameController.text = savedName;
    }
  }

  Future<void> _completeOnboarding() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setBool('onboarding_completed', true);
    await prefs.setString('device_name', _nameController.text.trim());
    widget.onFinished();
  }

  Future<void> _requestPermissions() async {
    try {
      await _channel.invokeMethod('getHeadphones');
      setState(() => _permissionsGranted = true);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Bluetooth permissions verified!')),
        );
      }
    } catch (_) {}
  }

  @override
  void dispose() {
    _nameController.dispose();
    _pageController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Baton First-Run Setup'),
      ),
      body: Column(
        children: [
          Expanded(
            child: PageView(
              controller: _pageController,
              onPageChanged: (idx) => setState(() => _currentPage = idx),
              children: [
                _buildStep(
                  icon: Icons.devices_other,
                  title: 'Name Your Device',
                  subtitle: 'Choose a friendly name so your computer can recognize this phone during audio handoffs.',
                  child: Column(
                    children: [
                      Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
                        child: TextField(
                          controller: _nameController,
                          decoration: const InputDecoration(
                            labelText: 'Device Name',
                            prefixIcon: Icon(Icons.edit),
                            border: OutlineInputBorder(),
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                      OutlinedButton.icon(
                        icon: Icon(
                          _permissionsGranted ? Icons.check_circle : Icons.bluetooth_searching,
                          color: _permissionsGranted ? Colors.greenAccent : null,
                        ),
                        label: Text(_permissionsGranted ? 'Permissions Granted' : 'Request Bluetooth Permissions'),
                        onPressed: _requestPermissions,
                      ),
                    ],
                  ),
                ),
                _buildStep(
                  icon: Icons.bluetooth_connected,
                  title: 'Crucial Prerequisite',
                  subtitle: 'Headphone Connection Requirement',
                  child: Padding(
                    padding: const EdgeInsets.all(16.0),
                    child: Card(
                      color: Colors.amber.shade900.withValues(alpha: 0.3),
                      child: const Padding(
                        padding: EdgeInsets.all(16.0),
                        child: Column(
                          children: [
                            Row(
                              children: [
                                Icon(Icons.warning_amber_rounded, color: Colors.amberAccent, size: 28),
                                SizedBox(width: 12),
                                Expanded(
                                  child: Text(
                                    'Pairing Requirement',
                                    style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                                  ),
                                ),
                              ],
                            ),
                            SizedBox(height: 8),
                            Text(
                              'Your Bluetooth headphones MUST be paired directly with BOTH your phone and your computer in your system Bluetooth Settings before using Baton.',
                              style: TextStyle(fontSize: 14, height: 1.4),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ),
                ),
                _buildStep(
                  icon: Icons.bolt,
                  title: 'Bluetooth Multipoint & Pairing',
                  subtitle: 'Instant Audio Handoff',
                  child: Column(
                    children: [
                      Padding(
                        padding: const EdgeInsets.all(16.0),
                        child: Card(
                          color: Colors.cyan.shade900.withValues(alpha: 0.3),
                          child: const Padding(
                            padding: EdgeInsets.all(16.0),
                            child: Row(
                              children: [
                                Icon(Icons.lightbulb_outline, color: Colors.cyanAccent, size: 28),
                                SizedBox(width: 12),
                                Expanded(
                                  child: Text(
                                    'If your headphones support Multipoint, Baton switches audio seamlessly without disconnecting!',
                                    style: TextStyle(fontSize: 14),
                                  ),
                                ),
                              ],
                            ),
                          ),
                        ),
                      ),
                      const SizedBox(height: 12),
                      ElevatedButton.icon(
                        icon: const Icon(Icons.qr_code_scanner),
                        label: const Text('Pair Computer Now (Scan QR)'),
                        onPressed: () {
                          Navigator.of(context).push(
                            MaterialPageRoute(builder: (_) => const PairingPage()),
                          );
                        },
                      ),
                    ],
                  ),
                ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.all(24.0),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                if (_currentPage > 0)
                  OutlinedButton(
                    onPressed: () {
                      _pageController.previousPage(
                        duration: const Duration(milliseconds: 300),
                        curve: Curves.easeInOut,
                      );
                    },
                    child: const Text('Back'),
                  )
                else
                  const SizedBox.shrink(),
                Row(
                  children: List.generate(
                    3,
                    (idx) => Container(
                      margin: const EdgeInsets.symmetric(horizontal: 4),
                      width: 8,
                      height: 8,
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: _currentPage == idx ? Colors.deepPurpleAccent : Colors.grey.shade700,
                      ),
                    ),
                  ),
                ),
                FilledButton(
                  onPressed: () {
                    if (_currentPage < 2) {
                      _pageController.nextPage(
                        duration: const Duration(milliseconds: 300),
                        curve: Curves.easeInOut,
                      );
                    } else {
                      _completeOnboarding();
                    }
                  },
                  child: Text(_currentPage == 2 ? 'Get Started' : 'Next'),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildStep({required IconData icon, required String title, required String subtitle, required Widget child}) {
    return Padding(
      padding: const EdgeInsets.all(24.0),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, size: 72, color: Colors.deepPurpleAccent),
          const SizedBox(height: 16),
          Text(title, style: const TextStyle(fontSize: 22, fontWeight: FontWeight.bold), textAlign: TextAlign.center),
          const SizedBox(height: 8),
          Text(subtitle, style: const TextStyle(fontSize: 14, color: Colors.grey), textAlign: TextAlign.center),
          const SizedBox(height: 20),
          child,
        ],
      ),
    );
  }
}
