import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../../core/models/device.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/feeding_provider.dart';

class ManualFeedScreen extends ConsumerStatefulWidget {
  const ManualFeedScreen({super.key});

  @override
  ConsumerState<ManualFeedScreen> createState() => _ManualFeedScreenState();
}

class _ManualFeedScreenState extends ConsumerState<ManualFeedScreen> {
  static const double _minAmount = 1;
  static const double _maxAmount = 500;

  double _amount =
      18.75; // 15 fish x 50g x 2.5% per feeding (5% BW/day / 2 feeds)
  String? _selectedDeviceId;
  late final TextEditingController _amountController;
  final FocusNode _amountFocusNode = FocusNode();

  @override
  void initState() {
    super.initState();
    _amountController = TextEditingController(text: _amount.toStringAsFixed(2));
    Future.microtask(_loadDevices);
  }

  @override
  void dispose() {
    _amountController.dispose();
    _amountFocusNode.dispose();
    super.dispose();
  }

  Future<void> _loadDevices() async {
    await ref.read(deviceListProvider.notifier).loadDevices();
    final devices = ref.read(devicesProvider);
    if (devices.isNotEmpty && _selectedDeviceId == null) {
      setState(() => _selectedDeviceId = devices.first.id);
    }
  }

  Future<void> _triggerFeed() async {
    if (_selectedDeviceId == null) return;

    FocusScope.of(context).unfocus();

    final success = await ref
        .read(manualFeedProvider.notifier)
        .triggerFeed(_selectedDeviceId!, _amount);

    if (mounted) {
      final state = ref.read(manualFeedProvider);
      if (success) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(state.successMessage ?? 'Feed command sent!')),
        );
      } else if (state.error != null) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(state.error!), backgroundColor: Colors.red),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    _syncAmountText();

    final deviceState = ref.watch(deviceListProvider);
    final feedState = ref.watch(manualFeedProvider);
    final selectedDevice =
        _selectedDeviceId != null
            ? ref.watch(deviceByIdProvider(_selectedDeviceId!))
            : null;

    return Scaffold(
      appBar: AppBar(title: const Text('Manual Feed')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            // Device selector
            Card(
              child: ListTile(
                leading: Icon(
                  Icons.router,
                  color:
                      selectedDevice?.isOnline == true
                          ? Colors.green
                          : Colors.grey,
                ),
                title: Text(selectedDevice?.name ?? 'Select a device'),
                subtitle: Text(
                  selectedDevice != null
                      ? '${selectedDevice.isOnline ? "Online" : "Offline"} • ${selectedDevice.status.feedLevel.toInt()}% feed remaining'
                      : 'No device selected',
                ),
                trailing: const Icon(Icons.arrow_drop_down),
                onTap: () => _showDeviceSelector(context, deviceState.devices),
              ),
            ),
            const SizedBox(height: 24),

            // Amount selector
            Card(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  children: [
                    Text(
                      'Feed Amount',
                      style: Theme.of(context).textTheme.titleMedium,
                    ),
                    const SizedBox(height: 16),
                    SizedBox(
                      width: 120,
                      child: TextFormField(
                        controller: _amountController,
                        focusNode: _amountFocusNode,
                        textAlign: TextAlign.right,
                        keyboardType: const TextInputType.numberWithOptions(
                          decimal: false,
                        ),
                        decoration: const InputDecoration(
                          isDense: true,
                          suffixText: 'g',
                          border: OutlineInputBorder(),
                        ),
                        onChanged: (text) {
                          final parsed = double.tryParse(text.trim());
                          if (parsed == null) return;
                          _setAmount(parsed);
                        },
                        onFieldSubmitted: (text) {
                          final parsed = double.tryParse(text.trim());
                          if (parsed == null) return;
                          _setAmount(parsed);
                        },
                      ),
                    ),
                    const SizedBox(height: 12),
                    Text(
                      '${_amount.round()}g',
                      style: Theme.of(
                        context,
                      ).textTheme.displayMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                        color: Theme.of(context).colorScheme.primary,
                      ),
                    ),
                    const SizedBox(height: 16),
                    Slider(
                      value: _amount,
                      min: _minAmount,
                      max: _maxAmount,
                      divisions: 18,
                      onChanged: _setAmount,
                    ),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(
                          '50g',
                          style: Theme.of(context).textTheme.bodySmall,
                        ),
                        Text(
                          '500g',
                          style: Theme.of(context).textTheme.bodySmall,
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),

            // Quick amounts
            Row(
              children: [
                _QuickAmountButton(
                  amount: 100,
                  isSelected: _amount == 100,
                  onTap: () => _setAmount(100),
                ),
                const SizedBox(width: 8),
                _QuickAmountButton(
                  amount: 200,
                  isSelected: _amount == 200,
                  onTap: () => _setAmount(200),
                ),
                const SizedBox(width: 8),
                _QuickAmountButton(
                  amount: 300,
                  isSelected: _amount == 300,
                  onTap: () => _setAmount(300),
                ),
                const SizedBox(width: 8),
                _QuickAmountButton(
                  amount: 400,
                  isSelected: _amount == 400,
                  onTap: () => _setAmount(400),
                ),
              ],
            ),

            if (selectedDevice != null && !selectedDevice.isOnline)
              Padding(
                padding: const EdgeInsets.only(top: 16),
                child: Card(
                  color: Colors.orange.shade50,
                  child: const Padding(
                    padding: EdgeInsets.all(12),
                    child: Row(
                      children: [
                        Icon(Icons.warning_amber, color: Colors.orange),
                        SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            'Device is offline. Command will be queued.',
                            style: TextStyle(color: Colors.orange),
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),

            const Spacer(),

            // Feed button
            SizedBox(
              height: 56,
              child: FilledButton.icon(
                onPressed:
                    feedState.isFeeding || _selectedDeviceId == null
                        ? null
                        : _triggerFeed,
                icon:
                    feedState.isFeeding
                        ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(
                            strokeWidth: 2,
                            color: Colors.white,
                          ),
                        )
                        : const Icon(Icons.restaurant),
                label: Text(feedState.isFeeding ? 'Dispensing...' : 'Feed Now'),
              ),
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
    );
  }

  void _setAmount(double value) {
    final clamped = value.clamp(_minAmount, _maxAmount).toDouble();
    if (clamped == _amount) {
      _syncAmountText();
      return;
    }

    setState(() {
      _amount = clamped;
    });
  }

  void _syncAmountText() {
    final text = _amount.round().toString();
    if (_amountFocusNode.hasFocus || _amountController.text == text) return;

    _amountController.value = _amountController.value.copyWith(
      text: text,
      selection: TextSelection.collapsed(offset: text.length),
      composing: TextRange.empty,
    );
  }

  void _showDeviceSelector(BuildContext context, List<Device> devices) {
    showModalBottomSheet(
      context: context,
      builder:
          (context) => ListView(
            shrinkWrap: true,
            children: [
              const Padding(
                padding: EdgeInsets.all(16),
                child: Text(
                  'Select Device',
                  style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                ),
              ),
              ...devices.map(
                (device) => ListTile(
                  leading: Icon(
                    Icons.router,
                    color: device.isOnline ? Colors.green : Colors.grey,
                  ),
                  title: Text(device.name),
                  subtitle: Text(
                    '${device.status.feedLevel.toInt()}% feed remaining',
                  ),
                  trailing:
                      _selectedDeviceId == device.id
                          ? const Icon(Icons.check, color: Colors.green)
                          : null,
                  onTap: () {
                    setState(() => _selectedDeviceId = device.id);
                    Navigator.pop(context);
                  },
                ),
              ),
            ],
          ),
    );
  }
}

class _QuickAmountButton extends StatelessWidget {
  final int amount;
  final bool isSelected;
  final VoidCallback onTap;

  const _QuickAmountButton({
    required this.amount,
    required this.isSelected,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Expanded(
      child:
          isSelected
              ? FilledButton(onPressed: onTap, child: Text('${amount}g'))
              : OutlinedButton(onPressed: onTap, child: Text('${amount}g')),
    );
  }
}
