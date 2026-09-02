import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../../../../core/services/ble_service.dart';
import '../../../../core/providers/device_provider.dart';

class DevicePairingScreen extends ConsumerStatefulWidget {
  const DevicePairingScreen({super.key});

  @override
  ConsumerState<DevicePairingScreen> createState() =>
      _DevicePairingScreenState();
}

class _DevicePairingScreenState extends ConsumerState<DevicePairingScreen> {
  final BleService _bleService = BleService();
  int _currentStep = 0;
  bool _isScanning = false;
  bool _isConnecting = false;
  bool _isProvisioning = false;
  String? _selectedDeviceId;
  String? _scannedSerialNumber;
  String _networkType = 'cellular';
  final _apnController = TextEditingController(text: 'internet');
  final _ssidController = TextEditingController();
  final _passwordController = TextEditingController();
  final _deviceNameController = TextEditingController(text: 'My Fish Feeder');
  final _manualSerialController = TextEditingController();
  final _manualCodeController = TextEditingController();
  List<BleDevice> _discoveredDevices = [];
  StreamSubscription? _scanSubscription;
  String? _bindingCode;
  String? _errorMessage;
  bool _showManualFields = false;

  bool get _canBindDirectly =>
      _selectedDeviceId == null && _isValidBindingCode(_bindingCode);

  bool _isValidBindingCode(String? code) {
    return code != null && RegExp(r'^\d{6}$').hasMatch(code);
  }

  @override
  void initState() {
    super.initState();
    _checkBluetooth();
    _setupBleListener();
  }

  @override
  void dispose() {
    _scanSubscription?.cancel();
    _bleService.disconnect();
    _apnController.dispose();
    _ssidController.dispose();
    _passwordController.dispose();
    _deviceNameController.dispose();
    _manualSerialController.dispose();
    _manualCodeController.dispose();
    super.dispose();
  }

  void _setupBleListener() {
    _scanSubscription = _bleService.discoveredDevices.listen((devices) {
      if (!mounted) return;
      setState(() => _discoveredDevices = devices);
    });
  }

  Future<void> _checkBluetooth() async {
    final isAvailable = await _bleService.isBluetoothAvailable();
    if (!isAvailable && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Bluetooth is not available on this device'),
        ),
      );
    }
  }

  Future<void> _startScan() async {
    setState(() {
      _isScanning = true;
      _discoveredDevices = [];
      _errorMessage = null;
    });

    try {
      await _bleService.startScan();
    } catch (e) {
      setState(() => _errorMessage = 'Scan failed: $e');
    } finally {
      if (mounted) {
        setState(() => _isScanning = false);
      }
    }
  }

  Future<void> _connectToDevice(BleDevice device) async {
    setState(() {
      _isConnecting = true;
      _errorMessage = null;
    });

    try {
      final success = await _bleService.connectToDevice(device.id);
      if (success) {
        setState(() {
          _selectedDeviceId = device.id;
          _currentStep = 2;
        });
      } else {
        setState(() => _errorMessage = 'Failed to connect to device');
      }
    } catch (e) {
      setState(() => _errorMessage = 'Connection error: $e');
    } finally {
      if (mounted) {
        setState(() => _isConnecting = false);
      }
    }
  }

  Future<void> _provisionDevice() async {
    setState(() {
      _isProvisioning = true;
      _errorMessage = null;
    });

    try {
      bool success;
      if (_networkType == 'cellular') {
        success = await _bleService.provisionCellular(_apnController.text);
      } else {
        success = await _bleService.provisionWifi(
          _ssidController.text,
          _passwordController.text,
        );
      }

      if (success) {
        // Get binding code from device
        _bindingCode = await _bleService.getBindingCode();

        if (_bindingCode == null) {
          setState(
            () =>
                _errorMessage =
                    'Could not retrieve binding code from device. Please try again.',
          );
          return;
        }

        // Bind device to user account
        final bindSuccess = await ref
            .read(deviceListProvider.notifier)
            .bindDevice(
              _scannedSerialNumber ?? '',
              _bindingCode!,
              _deviceNameController.text.trim().isEmpty
                  ? 'My Fish Feeder'
                  : _deviceNameController.text.trim(),
            );
        if (bindSuccess) {
          setState(() => _currentStep = 3);
        } else {
          setState(
            () => _errorMessage = 'Failed to bind device to your account',
          );
        }
      } else {
        setState(() => _errorMessage = 'Failed to provision device');
      }
    } catch (e) {
      setState(() => _errorMessage = 'Provisioning error: $e');
    } finally {
      if (mounted) {
        setState(() => _isProvisioning = false);
      }
    }
  }

  Future<void> _bindWithExistingCode() async {
    if (_scannedSerialNumber == null || !_isValidBindingCode(_bindingCode)) {
      setState(
        () =>
            _errorMessage = 'Enter the device serial and 6-digit binding code',
      );
      return;
    }

    setState(() {
      _isProvisioning = true;
      _errorMessage = null;
    });

    try {
      final bindSuccess = await ref
          .read(deviceListProvider.notifier)
          .bindDevice(
            _scannedSerialNumber!,
            _bindingCode!,
            _deviceNameController.text.trim().isEmpty
                ? 'My Fish Feeder'
                : _deviceNameController.text.trim(),
          );

      if (bindSuccess) {
        setState(() => _currentStep = 3);
      } else {
        setState(
          () =>
              _errorMessage =
                  'Failed to bind device. Check the serial number and binding code.',
        );
      }
    } catch (e) {
      setState(() => _errorMessage = 'Binding error: $e');
    } finally {
      if (mounted) {
        setState(() => _isProvisioning = false);
      }
    }
  }

  Future<void> _showQrScanner() async {
    var handled = false;
    final result = await showModalBottomSheet<Map<String, String?>>(
      context: context,
      isScrollControlled: true,
      builder:
          (ctx) => SizedBox(
            height: MediaQuery.of(ctx).size.height * 0.7,
            child: Column(
              children: [
                AppBar(
                  title: const Text('Scan QR Code'),
                  leading: IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: () => Navigator.pop(ctx),
                  ),
                ),
                Expanded(
                  child: MobileScanner(
                    onDetect: (capture) {
                      if (handled) return;
                      final barcodes = capture.barcodes;
                      if (barcodes.isNotEmpty) {
                        final code = barcodes.first.rawValue;
                        if (code == null) {
                          return;
                        }

                        String? serial;
                        String? scannedBindingCode;
                        if (code.startsWith('SFF-BIND|')) {
                          final parts = code.split('|');
                          if (parts.length >= 3) {
                            serial = parts[1];
                            scannedBindingCode = parts[2];
                          }
                        } else if (code.startsWith('SFF-')) {
                          serial = code;
                        }

                        if (serial != null) {
                          handled = true;
                          Navigator.pop(ctx, {
                            'serial': serial,
                            'bindingCode': scannedBindingCode,
                          });
                        }
                      }
                    },
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Text(
                    'Point camera at the QR code on your device',
                    style: TextStyle(color: Colors.grey[600]),
                    textAlign: TextAlign.center,
                  ),
                ),
              ],
            ),
          ),
    );

    if (!mounted || result == null) return;
    final serial = result['serial'];
    final bindingCode = result['bindingCode'];
    if (serial == null || serial.isEmpty) return;

    setState(() {
      _scannedSerialNumber = serial;
      _bindingCode = bindingCode;
      _currentStep = _isValidBindingCode(_bindingCode) ? 2 : 1;
    });
    if (!_isValidBindingCode(_bindingCode)) {
      unawaited(_startScan());
    }
  }

  void _showManualEntry() {
    _manualSerialController.text = _scannedSerialNumber ?? '';
    _manualCodeController.text = _bindingCode ?? '';
    setState(() {
      _showManualFields = true;
      _errorMessage = null;
    });
  }

  bool _applyManualEntry() {
    final serial = _manualSerialController.text.trim();
    final bindingCode = _manualCodeController.text.trim();

    if (serial.isEmpty) {
      setState(() => _errorMessage = 'Serial number is required');
      return false;
    }
    if (bindingCode.isNotEmpty && !_isValidBindingCode(bindingCode)) {
      setState(() => _errorMessage = 'Binding code must be 6 digits');
      return false;
    }

    FocusScope.of(context).unfocus();
    setState(() {
      _scannedSerialNumber = serial;
      _bindingCode = bindingCode.isEmpty ? null : bindingCode;
      _errorMessage = null;
    });
    return true;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Add Device')),
      body: SafeArea(
        child: Column(
          children: [
            _PairingStepHeader(currentStep: _currentStep, title: _stepTitle),
            Expanded(
              child: SingleChildScrollView(
                padding: const EdgeInsets.all(16),
                keyboardDismissBehavior:
                    ScrollViewKeyboardDismissBehavior.onDrag,
                child: _buildStepContent(),
              ),
            ),
            _buildBottomControls(),
          ],
        ),
      ),
    );
  }

  String get _stepTitle {
    switch (_currentStep) {
      case 0:
        return 'Identify Device';
      case 1:
        return 'Connect Device';
      case 2:
        return _canBindDirectly ? 'Bind Device' : 'Configure Network';
      default:
        return 'Complete Setup';
    }
  }

  Widget _buildStepContent() {
    switch (_currentStep) {
      case 0:
        return _buildIdentifyDeviceStep();
      case 1:
        return _buildConnectDeviceStep();
      case 2:
        return _buildConfigureDeviceStep();
      default:
        return _buildCompleteStep();
    }
  }

  Widget _buildIdentifyDeviceStep() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Scan the QR code on your device or enter the serial number manually.',
          style: TextStyle(color: Colors.grey),
        ),
        const SizedBox(height: 24),
        Row(
          children: [
            Expanded(
              child: _ActionCard(
                icon: Icons.qr_code_scanner,
                title: 'Scan QR Code',
                subtitle: 'Quick and easy',
                onTap: _showQrScanner,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: _ActionCard(
                icon: Icons.keyboard,
                title: 'Enter Manually',
                subtitle: 'Serial and code',
                onTap: _showManualEntry,
              ),
            ),
          ],
        ),
        if (_showManualFields) ...[
          const SizedBox(height: 16),
          TextFormField(
            controller: _manualSerialController,
            decoration: const InputDecoration(
              labelText: 'Serial Number',
              hintText: 'smartaqua-mcu',
              border: OutlineInputBorder(),
            ),
            textCapitalization: TextCapitalization.none,
            textInputAction: TextInputAction.next,
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _manualCodeController,
            decoration: const InputDecoration(
              labelText: 'Binding Code',
              hintText: '6 digits from Serial Monitor',
              border: OutlineInputBorder(),
            ),
            keyboardType: TextInputType.number,
            textInputAction: TextInputAction.done,
            inputFormatters: [
              FilteringTextInputFormatter.digitsOnly,
              LengthLimitingTextInputFormatter(6),
            ],
            onFieldSubmitted: (_) => _handleStepContinue(),
          ),
        ],
        if (_scannedSerialNumber != null) ...[
          const SizedBox(height: 16),
          Card(
            color: Colors.green.shade50,
            child: ListTile(
              leading: const Icon(Icons.check_circle, color: Colors.green),
              title: const Text('Device Found'),
              subtitle: Text(
                _bindingCode == null
                    ? _scannedSerialNumber!
                    : '$_scannedSerialNumber - Code $_bindingCode',
              ),
            ),
          ),
        ],
        if (_errorMessage != null) _buildErrorCard(),
      ],
    );
  }

  Widget _buildConnectDeviceStep() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (_scannedSerialNumber != null)
          Text('Looking for: $_scannedSerialNumber'),
        const SizedBox(height: 16),
        if (_isScanning)
          const Center(
            child: Column(
              children: [
                CircularProgressIndicator(),
                SizedBox(height: 16),
                Text('Scanning for nearby devices...'),
              ],
            ),
          )
        else if (_isConnecting)
          const Center(
            child: Column(
              children: [
                CircularProgressIndicator(),
                SizedBox(height: 16),
                Text('Connecting to device...'),
              ],
            ),
          )
        else ...[
          if (_discoveredDevices.isEmpty)
            Card(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  children: [
                    Icon(
                      Icons.bluetooth_searching,
                      size: 48,
                      color: Colors.grey[400],
                    ),
                    const SizedBox(height: 16),
                    const Text('No devices found'),
                    const SizedBox(height: 8),
                    const Text(
                      'Make sure your device is powered on and in pairing mode',
                      textAlign: TextAlign.center,
                      style: TextStyle(color: Colors.grey),
                    ),
                  ],
                ),
              ),
            )
          else
            ..._discoveredDevices.map(
              (device) => _DeviceListItem(
                name: device.name,
                signal: _getSignalStrength(device.rssi),
                isSelected: _selectedDeviceId == device.id,
                onTap: () => _connectToDevice(device),
              ),
            ),
          const SizedBox(height: 16),
          Center(
            child: OutlinedButton.icon(
              onPressed: _startScan,
              icon: const Icon(Icons.refresh),
              label: const Text('Scan Again'),
            ),
          ),
        ],
        if (_errorMessage != null) _buildErrorCard(),
      ],
    );
  }

  Widget _buildConfigureDeviceStep() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (_canBindDirectly) ...[
          const Text(
            'This device is already online. Bind it to your account with the code shown on the device serial monitor.',
            style: TextStyle(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 16),
          Card(
            child: ListTile(
              leading: const Icon(Icons.cloud_done),
              title: Text(_scannedSerialNumber ?? 'Device'),
              subtitle: Text('Binding code: $_bindingCode'),
            ),
          ),
        ] else ...[
          const Text(
            'Choose how your device connects to the internet:',
            style: TextStyle(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 16),
          _NetworkOption(
            icon: Icons.cell_tower,
            title: 'Cellular (Recommended)',
            subtitle: 'Uses SIM card for remote locations',
            isSelected: _networkType == 'cellular',
            onTap: () => setState(() => _networkType = 'cellular'),
          ),
          const SizedBox(height: 8),
          _NetworkOption(
            icon: Icons.wifi,
            title: 'WiFi',
            subtitle: 'Requires nearby WiFi network',
            isSelected: _networkType == 'wifi',
            onTap: () => setState(() => _networkType = 'wifi'),
          ),
          const SizedBox(height: 16),
          if (_networkType == 'cellular')
            TextFormField(
              controller: _apnController,
              decoration: const InputDecoration(
                labelText: 'APN',
                hintText: 'e.g., internet',
                border: OutlineInputBorder(),
              ),
            )
          else ...[
            TextFormField(
              controller: _ssidController,
              decoration: const InputDecoration(
                labelText: 'WiFi Network Name',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextFormField(
              controller: _passwordController,
              obscureText: true,
              decoration: const InputDecoration(
                labelText: 'WiFi Password',
                border: OutlineInputBorder(),
              ),
            ),
          ],
        ],
        const SizedBox(height: 16),
        TextFormField(
          controller: _deviceNameController,
          decoration: const InputDecoration(
            labelText: 'Device Name',
            hintText: 'e.g., Pond 1 Feeder',
            border: OutlineInputBorder(),
          ),
        ),
        if (_isProvisioning) ...[
          const SizedBox(height: 24),
          const Center(
            child: Column(
              children: [
                CircularProgressIndicator(),
                SizedBox(height: 16),
                Text('Working...'),
              ],
            ),
          ),
        ],
        if (_errorMessage != null) _buildErrorCard(),
      ],
    );
  }

  Widget _buildCompleteStep() {
    return const Center(
      child: Column(
        children: [
          Icon(Icons.check_circle, color: Colors.green, size: 64),
          SizedBox(height: 16),
          Text(
            'Device Added Successfully!',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
          ),
          SizedBox(height: 8),
          Text(
            'Your SmartAqua feeder is now connected and ready to use.',
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  Widget _buildErrorCard() {
    return Padding(
      padding: const EdgeInsets.only(top: 16),
      child: Card(
        color: Colors.red.shade50,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            children: [
              const Icon(Icons.error, color: Colors.red),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  _errorMessage!,
                  style: const TextStyle(color: Colors.red),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildBottomControls() {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: Theme.of(context).scaffoldBackgroundColor,
        border: Border(top: BorderSide(color: Theme.of(context).dividerColor)),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
        child: Row(
          children: [
            if (_currentStep < 3)
              FilledButton(
                onPressed: _canContinue() ? _handleStepContinue : null,
                child: _getButtonChild(),
              ),
            if (_currentStep == 3)
              FilledButton(
                onPressed: () => context.go('/devices'),
                child: const Text('Done'),
              ),
            const SizedBox(width: 12),
            if (_currentStep > 0 && _currentStep < 3)
              TextButton(
                onPressed: _handleStepCancel,
                child: const Text('Back'),
              ),
          ],
        ),
      ),
    );
  }

  bool _canContinue() {
    switch (_currentStep) {
      case 0:
        return _scannedSerialNumber != null || _showManualFields;
      case 1:
        return _selectedDeviceId != null && !_isConnecting;
      case 2:
        return !_isProvisioning &&
            (!_canBindDirectly || _scannedSerialNumber != null);
      default:
        return true;
    }
  }

  Widget _getButtonChild() {
    if (_currentStep == 2 && _isProvisioning) {
      return const SizedBox(
        width: 20,
        height: 20,
        child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
      );
    }
    return Text(
      _currentStep == 2
          ? (_canBindDirectly ? 'Bind Device' : 'Configure')
          : 'Continue',
    );
  }

  void _handleStepContinue() {
    if (_currentStep == 0) {
      if (_showManualFields && !_applyManualEntry()) {
        return;
      }
      if (_isValidBindingCode(_bindingCode)) {
        setState(() => _currentStep = 2);
      } else {
        setState(() => _currentStep = 1);
        unawaited(_startScan());
      }
    } else if (_currentStep == 2) {
      if (_canBindDirectly) {
        _bindWithExistingCode();
      } else {
        _provisionDevice();
      }
    } else if (_currentStep < 3) {
      setState(() => _currentStep++);
    }
  }

  void _handleStepCancel() {
    if (_currentStep > 0) {
      setState(() => _currentStep--);
    } else {
      context.pop();
    }
  }

  String _getSignalStrength(int rssi) {
    if (rssi >= -50) return 'Excellent';
    if (rssi >= -60) return 'Good';
    if (rssi >= -70) return 'Fair';
    return 'Weak';
  }
}

class _PairingStepHeader extends StatelessWidget {
  final int currentStep;
  final String title;

  const _PairingStepHeader({required this.currentStep, required this.title});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Step ${currentStep + 1} of 4',
            style: theme.textTheme.labelLarge?.copyWith(
              color: theme.colorScheme.primary,
            ),
          ),
          const SizedBox(height: 4),
          Text(title, style: theme.textTheme.headlineSmall),
          const SizedBox(height: 12),
          ClipRRect(
            borderRadius: BorderRadius.circular(4),
            child: LinearProgressIndicator(
              minHeight: 6,
              value: (currentStep + 1) / 4,
            ),
          ),
        ],
      ),
    );
  }
}

class _ActionCard extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  const _ActionCard({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              Icon(
                icon,
                size: 40,
                color: Theme.of(context).colorScheme.primary,
              ),
              const SizedBox(height: 8),
              Text(title, style: const TextStyle(fontWeight: FontWeight.bold)),
              Text(
                subtitle,
                style: TextStyle(fontSize: 12, color: Colors.grey[600]),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _DeviceListItem extends StatelessWidget {
  final String name;
  final String signal;
  final bool isSelected;
  final VoidCallback onTap;

  const _DeviceListItem({
    required this.name,
    required this.signal,
    required this.isSelected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      color: isSelected ? Theme.of(context).colorScheme.primaryContainer : null,
      child: ListTile(
        leading: const Icon(Icons.bluetooth),
        title: Text(name),
        subtitle: Text('Signal: $signal'),
        trailing:
            isSelected
                ? const Icon(Icons.check_circle, color: Colors.green)
                : null,
        onTap: onTap,
      ),
    );
  }
}

class _NetworkOption extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final bool isSelected;
  final VoidCallback onTap;

  const _NetworkOption({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.isSelected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      color: isSelected ? Theme.of(context).colorScheme.primaryContainer : null,
      child: ListTile(
        leading: Icon(icon),
        title: Text(title),
        subtitle: Text(subtitle),
        trailing:
            isSelected
                ? const Icon(Icons.radio_button_checked, color: Colors.green)
                : const Icon(Icons.radio_button_off),
        onTap: onTap,
      ),
    );
  }
}
