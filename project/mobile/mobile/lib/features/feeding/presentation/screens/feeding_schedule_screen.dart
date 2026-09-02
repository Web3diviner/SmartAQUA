import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/device.dart';
import '../../../../core/models/feeding.dart';
import '../../../../core/providers/calculator_provider.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/feeding_provider.dart';

class FeedingScheduleScreen extends ConsumerStatefulWidget {
  const FeedingScheduleScreen({super.key});

  @override
  ConsumerState<FeedingScheduleScreen> createState() =>
      _FeedingScheduleScreenState();
}

class _FeedingScheduleScreenState extends ConsumerState<FeedingScheduleScreen> {
  String? _selectedDeviceId;

  _CalculatorSuggestion? _latestCalculatorSuggestion() {
    final calculatorState = ref.read(calculatorProvider);
    final result = calculatorState.result;
    if (result == null) return null;

    const suggestedFeedings = 2;
    final perFeeding = result.recommendedAmount / suggestedFeedings;
    final request = calculatorState.lastRequest;

    return _CalculatorSuggestion(
      recommendedDailyAmount: result.recommendedAmount,
      recommendedPerFeedingAmount: perFeeding,
      suggestedFeedings: suggestedFeedings,
      calculatedAt: calculatorState.lastCalculatedAt ?? DateTime.now(),
      waterTemperature: request?.waterTemperature,
    );
  }

  @override
  void initState() {
    super.initState();
    Future.microtask(_loadData);
  }

  Future<void> _loadData() async {
    await ref.read(deviceListProvider.notifier).loadDevices();
    final devices = ref.read(devicesProvider);
    if (devices.isNotEmpty && _selectedDeviceId == null) {
      _selectedDeviceId = devices.first.id;
      await ref
          .read(feedingSchedulesProvider.notifier)
          .loadSchedules(_selectedDeviceId!);
    }
  }

  @override
  Widget build(BuildContext context) {
    final deviceState = ref.watch(deviceListProvider);
    final schedulesState = ref.watch(feedingSchedulesProvider);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Feeding Schedules'),
        actions: [
          IconButton(
            icon: const Icon(Icons.calculate_outlined),
            tooltip: 'Feed Calculator',
            onPressed: () => context.go('/calculator'),
          ),
          IconButton(
            icon: const Icon(Icons.history),
            onPressed: () => context.go('/feeding/history'),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async {
          if (_selectedDeviceId != null) {
            await ref
                .read(feedingSchedulesProvider.notifier)
                .loadSchedules(_selectedDeviceId!);
          }
        },
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            // Device selector
            Card(
              child: ListTile(
                leading: const Icon(Icons.router),
                title: Text(_getSelectedDeviceName(deviceState.devices)),
                subtitle: Text(_selectedDeviceId ?? 'Select a device'),
                trailing: const Icon(Icons.arrow_drop_down),
                onTap: () => _showDeviceSelector(context, deviceState.devices),
              ),
            ),
            const SizedBox(height: 16),

            if (schedulesState.isLoading)
              const Center(child: CircularProgressIndicator())
            else if (schedulesState.error != null)
              Center(child: Text(schedulesState.error!))
            else if (schedulesState.schedules.isEmpty)
              _buildEmptyState(context)
            else
              ...schedulesState.schedules.map(
                (schedule) => _ScheduleCard(
                  schedule: schedule,
                  onToggle: () => _toggleSchedule(schedule),
                  onEdit: () => _showEditScheduleDialog(context, schedule),
                  onDelete: () => _deleteSchedule(schedule),
                ),
              ),
          ],
        ),
      ),
      floatingActionButton: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          FloatingActionButton(
            heroTag: 'manual',
            onPressed: () => context.go('/feeding/manual'),
            child: const Icon(Icons.restaurant),
          ),
          const SizedBox(height: 12),
          FloatingActionButton.extended(
            heroTag: 'schedule',
            onPressed:
                _selectedDeviceId != null
                    ? () => _showAddScheduleDialog(context)
                    : null,
            icon: const Icon(Icons.add),
            label: const Text('Add Schedule'),
          ),
        ],
      ),
    );
  }

  String _getSelectedDeviceName(List<Device> devices) {
    if (_selectedDeviceId == null) return 'No device selected';
    try {
      return devices.firstWhere((d) => d.id == _selectedDeviceId).name;
    } catch (_) {
      return 'Unknown device';
    }
  }

  Widget _buildEmptyState(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          children: [
            Icon(Icons.schedule, size: 48, color: Colors.grey[400]),
            const SizedBox(height: 16),
            Text(
              'No schedules yet',
              style: Theme.of(
                context,
              ).textTheme.titleMedium?.copyWith(color: Colors.grey[600]),
            ),
            const SizedBox(height: 8),
            Text(
              'Add a feeding schedule to automate feeding',
              style: TextStyle(color: Colors.grey[500]),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
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
                  subtitle: Text(device.serialNumber),
                  trailing:
                      _selectedDeviceId == device.id
                          ? const Icon(Icons.check, color: Colors.green)
                          : null,
                  onTap: () {
                    setState(() => _selectedDeviceId = device.id);
                    ref
                        .read(feedingSchedulesProvider.notifier)
                        .loadSchedules(device.id);
                    Navigator.pop(context);
                  },
                ),
              ),
            ],
          ),
    );
  }

  void _showAddScheduleDialog(BuildContext context) {
    final suggestion = _latestCalculatorSuggestion();

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder:
          (ctx) => _AddScheduleSheet(
            deviceId: _selectedDeviceId!,
            calculatorSuggestion: suggestion,
            onSave: (schedule) async {
              final success = await ref
                  .read(feedingSchedulesProvider.notifier)
                  .createSchedule(_selectedDeviceId!, schedule);
              if (success && ctx.mounted) {
                ScaffoldMessenger.of(ctx).showSnackBar(
                  const SnackBar(content: Text('Schedule created')),
                );
              }
            },
          ),
    );
  }

  void _showEditScheduleDialog(BuildContext context, FeedingSchedule schedule) {
    final suggestion = _latestCalculatorSuggestion();

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      builder:
          (ctx) => _AddScheduleSheet(
            deviceId: _selectedDeviceId!,
            schedule: schedule,
            calculatorSuggestion: suggestion,
            onSave: (updated) async {
              final success = await ref
                  .read(feedingSchedulesProvider.notifier)
                  .updateSchedule(_selectedDeviceId!, updated);
              if (success && ctx.mounted) {
                ScaffoldMessenger.of(ctx).showSnackBar(
                  const SnackBar(content: Text('Schedule updated')),
                );
              }
            },
          ),
    );
  }

  Future<void> _toggleSchedule(FeedingSchedule schedule) async {
    await ref
        .read(feedingSchedulesProvider.notifier)
        .toggleSchedule(_selectedDeviceId!, schedule);
  }

  Future<void> _deleteSchedule(FeedingSchedule schedule) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder:
          (context) => AlertDialog(
            title: const Text('Delete Schedule'),
            content: const Text(
              'Are you sure you want to delete this schedule?',
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(context, false),
                child: const Text('Cancel'),
              ),
              TextButton(
                onPressed: () => Navigator.pop(context, true),
                child: const Text(
                  'Delete',
                  style: TextStyle(color: Colors.red),
                ),
              ),
            ],
          ),
    );

    if (confirm == true) {
      final success = await ref
          .read(feedingSchedulesProvider.notifier)
          .deleteSchedule(_selectedDeviceId!, schedule.id);
      if (success && mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('Schedule deleted')));
      }
    }
  }
}

class _ScheduleCard extends StatelessWidget {
  final FeedingSchedule schedule;
  final VoidCallback onToggle;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  const _ScheduleCard({
    required this.schedule,
    required this.onToggle,
    required this.onEdit,
    required this.onDelete,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          children: [
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    schedule.time,
                    style: Theme.of(context).textTheme.titleLarge?.copyWith(
                      fontWeight: FontWeight.bold,
                      color: schedule.isEnabled ? null : Colors.grey,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${schedule.amount.toInt()}g • ${schedule.daysDescription}',
                    style: Theme.of(
                      context,
                    ).textTheme.bodyMedium?.copyWith(color: Colors.grey),
                  ),
                ],
              ),
            ),
            Switch(value: schedule.isEnabled, onChanged: (_) => onToggle()),
            PopupMenuButton<String>(
              onSelected: (value) {
                if (value == 'edit') onEdit();
                if (value == 'delete') onDelete();
              },
              itemBuilder:
                  (context) => [
                    const PopupMenuItem(value: 'edit', child: Text('Edit')),
                    const PopupMenuItem(
                      value: 'delete',
                      child: Text(
                        'Delete',
                        style: TextStyle(color: Colors.red),
                      ),
                    ),
                  ],
            ),
          ],
        ),
      ),
    );
  }
}

class _AddScheduleSheet extends StatefulWidget {
  final String deviceId;
  final FeedingSchedule? schedule;
  final _CalculatorSuggestion? calculatorSuggestion;
  final Function(FeedingSchedule) onSave;

  const _AddScheduleSheet({
    required this.deviceId,
    this.schedule,
    this.calculatorSuggestion,
    required this.onSave,
  });

  @override
  State<_AddScheduleSheet> createState() => _AddScheduleSheetState();
}

class _AddScheduleSheetState extends State<_AddScheduleSheet> {
  static const double _minAmount = 10;
  static const double _maxAmount = 500;

  late TimeOfDay _selectedTime;
  late double _amount;
  late Set<int> _selectedDays;
  late final TextEditingController _amountController;
  final FocusNode _amountFocusNode = FocusNode();
  bool _autoFilledFromCalculator = false;

  @override
  void initState() {
    super.initState();
    if (widget.schedule != null) {
      final parts = widget.schedule!.time.split(':');
      _selectedTime = TimeOfDay(
        hour: int.parse(parts[0]),
        minute: int.parse(parts[1]),
      );
      _amount = widget.schedule!.amount;
      _selectedDays = widget.schedule!.daysOfWeek.toSet();
    } else if (widget.calculatorSuggestion != null) {
      _selectedTime = const TimeOfDay(hour: 8, minute: 0);
      _amount = _clampAmount(
        widget.calculatorSuggestion!.recommendedPerFeedingAmount,
      );
      _selectedDays = {0, 1, 2, 3, 4, 5, 6};
      _autoFilledFromCalculator = true;
    } else {
      _selectedTime = const TimeOfDay(hour: 8, minute: 0);
      _amount = 18.75; // 15 fish x 50g x 2.5% per feeding (5% BW/day / 2 feeds)
      _selectedDays = {0, 1, 2, 3, 4, 5, 6};
    }

    _amountController = TextEditingController(text: _amount.toStringAsFixed(2));
  }

  @override
  void dispose() {
    _amountController.dispose();
    _amountFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    _syncAmountText();

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
        left: 16,
        right: 16,
        top: 16,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            widget.schedule != null ? 'Edit Schedule' : 'Add Schedule',
            style: Theme.of(
              context,
            ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 24),

          ListTile(
            leading: const Icon(Icons.access_time),
            title: const Text('Time'),
            trailing: Text(
              _selectedTime.format(context),
              style: Theme.of(context).textTheme.titleMedium,
            ),
            onTap: () async {
              final time = await showTimePicker(
                context: context,
                initialTime: _selectedTime,
              );
              if (time != null) {
                setState(() => _selectedTime = time);
              }
            },
          ),
          const Divider(),

          ListTile(
            leading: const Icon(Icons.scale),
            title: const Text('Amount'),
            subtitle: Slider(
              value: _amount,
              min: _minAmount,
              max: _maxAmount,
              divisions: 18,
              label: '${_amount.round()}g',
              onChanged: _setAmount,
            ),
            trailing: SizedBox(
              width: 88,
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
          ),
          if (widget.calculatorSuggestion != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Card(
                color: Theme.of(context).colorScheme.surfaceContainerHighest,
                child: Padding(
                  padding: const EdgeInsets.all(12),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        _calculatorSummary(widget.calculatorSuggestion!),
                        style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        _calculatorContext(widget.calculatorSuggestion!),
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                      const SizedBox(height: 8),
                      FilledButton.tonalIcon(
                        onPressed: _applyCalculatorSuggestion,
                        icon: const Icon(Icons.auto_awesome),
                        label: Text(
                          _autoFilledFromCalculator
                              ? 'Reapply calculator amount'
                              : 'Use calculator amount',
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          const Divider(),

          const ListTile(
            leading: Icon(Icons.calendar_today),
            title: Text('Repeat'),
          ),
          Wrap(
            spacing: 8,
            children: List.generate(7, (index) {
              final days = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'];
              final isSelected = _selectedDays.contains(index);
              return FilterChip(
                label: Text(days[index]),
                selected: isSelected,
                onSelected: (selected) {
                  setState(() {
                    if (selected) {
                      _selectedDays.add(index);
                    } else {
                      _selectedDays.remove(index);
                    }
                  });
                },
              );
            }),
          ),
          const SizedBox(height: 24),

          FilledButton(
            onPressed: _selectedDays.isNotEmpty ? _save : null,
            child: Text(
              widget.schedule != null ? 'Update Schedule' : 'Save Schedule',
            ),
          ),
          const SizedBox(height: 16),
        ],
      ),
    );
  }

  void _setAmount(double value) {
    final clamped = _clampAmount(value);
    if (clamped == _amount) {
      _syncAmountText();
      return;
    }

    setState(() {
      _amount = clamped;
      _autoFilledFromCalculator = false;
    });
  }

  double _clampAmount(double value) {
    return value.clamp(_minAmount, _maxAmount).toDouble();
  }

  void _applyCalculatorSuggestion() {
    final suggestion = widget.calculatorSuggestion;
    if (suggestion == null) return;
    setState(() {
      _amount = _clampAmount(suggestion.recommendedPerFeedingAmount);
      _autoFilledFromCalculator = true;
    });
  }

  String _calculatorSummary(_CalculatorSuggestion suggestion) {
    return 'Calculator: ${suggestion.recommendedDailyAmount.round()}g/day -> '
        '${suggestion.recommendedPerFeedingAmount.round()}g/feed '
        '(${suggestion.suggestedFeedings}x/day)';
  }

  String _calculatorContext(_CalculatorSuggestion suggestion) {
    final temp = suggestion.waterTemperature;
    final parts = <String>[];
    if (temp != null) {
      parts.add('Temp ${temp.toStringAsFixed(1)}°C');
    }

    final contextLine =
        parts.isEmpty
            ? 'Based on your last calculator run.'
            : '${parts.join(' • ')} from your last calculator run.';

    final time =
        '${suggestion.calculatedAt.hour.toString().padLeft(2, '0')}:${suggestion.calculatedAt.minute.toString().padLeft(2, '0')}';
    return '$contextLine ($time)';
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

  void _save() {
    final timeStr =
        '${_selectedTime.hour.toString().padLeft(2, '0')}:${_selectedTime.minute.toString().padLeft(2, '0')}';
    final schedule = FeedingSchedule(
      id: widget.schedule?.id ?? '',
      deviceId: widget.deviceId,
      time: timeStr,
      amount: _amount,
      durationSeconds: widget.schedule?.durationSeconds ?? 10,
      daysOfWeek: _selectedDays.toList()..sort(),
      isEnabled: widget.schedule?.isEnabled ?? true,
    );
    widget.onSave(schedule);
    Navigator.pop(context);
  }
}

class _CalculatorSuggestion {
  final double recommendedDailyAmount;
  final double recommendedPerFeedingAmount;
  final int suggestedFeedings;
  final DateTime calculatedAt;
  final double? waterTemperature;

  const _CalculatorSuggestion({
    required this.recommendedDailyAmount,
    required this.recommendedPerFeedingAmount,
    required this.suggestedFeedings,
    required this.calculatedAt,
    this.waterTemperature,
  });
}
