import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/device.dart';
import '../../../../core/models/farm_unit.dart';
import '../../../../core/models/feeding.dart';
import '../../../../core/providers/calculator_provider.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/farm_provider.dart';
import '../../../../core/providers/feeding_provider.dart';
import '../../../../core/theme/app_theme.dart';

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
    final farmUnits = ref.read(farmUnitsProvider).units;

    if (_selectedDeviceId == null) {
      if (devices.isNotEmpty) {
        _selectedDeviceId = devices.first.id;
      } else if (farmUnits.isNotEmpty) {
        _selectedDeviceId = farmUnits.first.id;
      } else {
        _selectedDeviceId = 'SFF-001';
      }
      await ref
          .read(feedingSchedulesProvider.notifier)
          .loadSchedules(_selectedDeviceId!);
    }
  }

  String _getSelectedDeviceName(List<Device> devices, List<FarmUnit> units) {
    final id = _selectedDeviceId ?? 'SFF-001';
    final deviceMatch = devices.where((d) => d.id == id).firstOrNull;
    if (deviceMatch != null) return deviceMatch.name;

    final unitMatch = units.where((u) => u.id == id).firstOrNull;
    if (unitMatch != null) return 'Pond: ${unitMatch.name}';

    if (id == 'SFF-001') return 'Smart Feeder #1 (Default)';
    return 'Feeder Node ($id)';
  }

  @override
  Widget build(BuildContext context) {
    final deviceState = ref.watch(deviceListProvider);
    final farmUnits = ref.watch(farmUnitsProvider).units;
    final schedulesState = ref.watch(feedingSchedulesProvider);

    // Auto-resolve device id if unset
    if (_selectedDeviceId == null) {
      if (deviceState.devices.isNotEmpty) {
        _selectedDeviceId = deviceState.devices.first.id;
      } else if (farmUnits.isNotEmpty) {
        _selectedDeviceId = farmUnits.first.id;
      } else {
        _selectedDeviceId = 'SFF-001';
      }
    }

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
            tooltip: 'Feeding History',
            onPressed: () => context.go('/feeding/history'),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async {
          final id = _selectedDeviceId ?? 'SFF-001';
          await ref.read(feedingSchedulesProvider.notifier).loadSchedules(id);
        },
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            // Device / Pond Selector Card
            Card(
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
              child: ListTile(
                leading: Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: Theme.of(context).colorScheme.primary.withValues(alpha: 0.15),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Icon(
                    Icons.router,
                    color: Theme.of(context).colorScheme.primary,
                  ),
                ),
                title: Text(
                  _getSelectedDeviceName(deviceState.devices, farmUnits),
                  style: const TextStyle(fontWeight: FontWeight.bold),
                ),
                subtitle: Text('Target: ${_selectedDeviceId ?? "SFF-001"}'),
                trailing: const Icon(Icons.arrow_drop_down),
                onTap: () => _showDeviceSelector(context, deviceState.devices, farmUnits),
              ),
            ),
            const SizedBox(height: 16),

            if (schedulesState.isLoading)
              const Padding(
                padding: EdgeInsets.all(40),
                child: Center(child: CircularProgressIndicator()),
              )
            else if (schedulesState.error != null)
              Center(
                child: Padding(
                  padding: const EdgeInsets.all(24),
                  child: Text(schedulesState.error!),
                ),
              )
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
            const SizedBox(height: 80), // Padding for FAB
          ],
        ),
      ),
      floatingActionButton: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          FloatingActionButton.small(
            heroTag: 'manual',
            tooltip: 'Manual Dispense',
            onPressed: () => context.go('/feeding/manual'),
            child: const Icon(Icons.restaurant),
          ),
          const SizedBox(height: 12),
          FloatingActionButton.extended(
            heroTag: 'schedule',
            onPressed: () => _showAddScheduleDialog(context),
            icon: const Icon(Icons.add),
            label: const Text('Add Schedule', style: TextStyle(fontWeight: FontWeight.bold)),
          ),
        ],
      ),
    );
  }

  Widget _buildEmptyState(BuildContext context) {
    return Card(
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(18)),
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          children: [
            Icon(Icons.schedule, size: 56, color: Colors.grey[400]),
            const SizedBox(height: 16),
            Text(
              'No Feeding Schedules Configured',
              style: Theme.of(context).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 8),
            Text(
              'Automate timed feed rations for your fish to maintain optimal SGR and avoid water fouling.',
              style: TextStyle(color: Colors.grey[500], fontSize: 13),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 20),
            FilledButton.icon(
              onPressed: () => _showAddScheduleDialog(context),
              icon: const Icon(Icons.add),
              label: const Text('Add First Schedule'),
            ),
          ],
        ),
      ),
    );
  }

  void _showDeviceSelector(BuildContext context, List<Device> devices, List<FarmUnit> farmUnits) {
    showModalBottomSheet(
      context: context,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      builder: (ctx) => SafeArea(
        child: ListView(
          shrinkWrap: true,
          padding: const EdgeInsets.symmetric(vertical: 12),
          children: [
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
              child: Text(
                'Select Feeder or Farm Pond',
                style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
              ),
            ),
            const Divider(),
            if (devices.isNotEmpty) ...[
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                child: Text('Hardware Feeder Nodes', style: TextStyle(color: Colors.grey, fontSize: 12, fontWeight: FontWeight.bold)),
              ),
              ...devices.map(
                (device) => ListTile(
                  leading: Icon(
                    Icons.router,
                    color: device.isOnline ? Colors.green : Colors.grey,
                  ),
                  title: Text(device.name),
                  subtitle: Text('ID: ${device.id} • ${device.isOnline ? "Online" : "Offline"}'),
                  trailing: _selectedDeviceId == device.id ? const Icon(Icons.check_circle, color: Colors.green) : null,
                  onTap: () {
                    setState(() => _selectedDeviceId = device.id);
                    ref.read(feedingSchedulesProvider.notifier).loadSchedules(device.id);
                    Navigator.pop(ctx);
                  },
                ),
              ),
              const Divider(),
            ],
            if (farmUnits.isNotEmpty) ...[
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 16, vertical: 4),
                child: Text('Farm Ponds / Units', style: TextStyle(color: Colors.grey, fontSize: 12, fontWeight: FontWeight.bold)),
              ),
              ...farmUnits.map(
                (unit) => ListTile(
                  leading: const Icon(Icons.waves, color: Colors.cyanAccent),
                  title: Text(unit.name),
                  subtitle: Text('${unit.fishCount} fish • ${unit.volumeM3.toStringAsFixed(0)} m³'),
                  trailing: _selectedDeviceId == unit.id ? const Icon(Icons.check_circle, color: Colors.green) : null,
                  onTap: () {
                    setState(() => _selectedDeviceId = unit.id);
                    ref.read(feedingSchedulesProvider.notifier).loadSchedules(unit.id);
                    Navigator.pop(ctx);
                  },
                ),
              ),
              const Divider(),
            ],
            ListTile(
              leading: const Icon(Icons.memory, color: Colors.tealAccent),
              title: const Text('Smart Feeder #1 (Default Simulation)'),
              subtitle: const Text('Default automation feeder node'),
              trailing: _selectedDeviceId == 'SFF-001' ? const Icon(Icons.check_circle, color: Colors.green) : null,
              onTap: () {
                setState(() => _selectedDeviceId = 'SFF-001');
                ref.read(feedingSchedulesProvider.notifier).loadSchedules('SFF-001');
                Navigator.pop(ctx);
              },
            ),
          ],
        ),
      ),
    );
  }

  void _showAddScheduleDialog(BuildContext context) {
    final suggestion = _latestCalculatorSuggestion();
    final activeId = _selectedDeviceId ?? 'SFF-001';
    final messenger = ScaffoldMessenger.of(context);

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (ctx) => _AddScheduleSheet(
        deviceId: activeId,
        calculatorSuggestion: suggestion,
        onSave: (schedule) async {
          await ref.read(feedingSchedulesProvider.notifier).createSchedule(activeId, schedule);
          messenger.showSnackBar(
            SnackBar(
              content: Text('Added feeding schedule for ${schedule.time} (${schedule.amount.toInt()}g)'),
              backgroundColor: AppTheme.deviceOnline,
            ),
          );
        },
      ),
    );
  }

  void _showEditScheduleDialog(BuildContext context, FeedingSchedule schedule) {
    final suggestion = _latestCalculatorSuggestion();
    final activeId = _selectedDeviceId ?? 'SFF-001';
    final messenger = ScaffoldMessenger.of(context);

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (ctx) => _AddScheduleSheet(
        deviceId: activeId,
        schedule: schedule,
        calculatorSuggestion: suggestion,
        onSave: (updated) async {
          await ref.read(feedingSchedulesProvider.notifier).updateSchedule(activeId, updated);
          messenger.showSnackBar(
            SnackBar(
              content: Text('Updated feeding schedule for ${updated.time} (${updated.amount.toInt()}g)'),
              backgroundColor: AppTheme.deviceOnline,
            ),
          );
        },
      ),
    );
  }

  Future<void> _toggleSchedule(FeedingSchedule schedule) async {
    final activeId = _selectedDeviceId ?? 'SFF-001';
    await ref.read(feedingSchedulesProvider.notifier).toggleSchedule(activeId, schedule);
  }

  Future<void> _deleteSchedule(FeedingSchedule schedule) async {
    final confirm = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete Feeding Schedule'),
        content: Text('Remove schedule for ${schedule.time} (${schedule.amount.toInt()}g)?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.pop(context, true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );

    if (confirm == true) {
      final activeId = _selectedDeviceId ?? 'SFF-001';
      await ref.read(feedingSchedulesProvider.notifier).deleteSchedule(activeId, schedule.id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Schedule removed')),
        );
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
    final isDark = Theme.of(context).brightness == Brightness.dark;

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(
          color: schedule.isEnabled
              ? Theme.of(context).colorScheme.primary.withValues(alpha: 0.3)
              : Colors.transparent,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        child: Row(
          children: [
            Container(
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: (schedule.isEnabled
                        ? Theme.of(context).colorScheme.primary
                        : Colors.grey)
                    .withValues(alpha: 0.15),
                borderRadius: BorderRadius.circular(12),
              ),
              child: Icon(
                Icons.access_time_filled,
                color: schedule.isEnabled
                    ? Theme.of(context).colorScheme.primary
                    : Colors.grey,
                size: 24,
              ),
            ),
            const SizedBox(width: 14),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Text(
                        schedule.time,
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                              fontWeight: FontWeight.bold,
                              color: schedule.isEnabled
                                  ? (isDark ? Colors.white : Colors.black87)
                                  : Colors.grey,
                            ),
                      ),
                      const SizedBox(width: 8),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                        decoration: BoxDecoration(
                          color: Theme.of(context).colorScheme.primary.withValues(alpha: 0.12),
                          borderRadius: BorderRadius.circular(8),
                        ),
                        child: Text(
                          '${schedule.amount.toInt()}g',
                          style: TextStyle(
                            color: Theme.of(context).colorScheme.primary,
                            fontWeight: FontWeight.bold,
                            fontSize: 12,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 4),
                  Text(
                    schedule.daysDescription,
                    style: TextStyle(color: Colors.grey[500], fontSize: 12),
                  ),
                ],
              ),
            ),
            Switch(
              value: schedule.isEnabled,
              activeThumbColor: Theme.of(context).colorScheme.primary,
              onChanged: (_) => onToggle(),
            ),
            PopupMenuButton<String>(
              icon: const Icon(Icons.more_vert, size: 20),
              onSelected: (value) {
                if (value == 'edit') onEdit();
                if (value == 'delete') onDelete();
              },
              itemBuilder: (context) => [
                const PopupMenuItem(
                  value: 'edit',
                  child: Row(
                    children: [
                      Icon(Icons.edit, size: 18),
                      SizedBox(width: 8),
                      Text('Edit Schedule'),
                    ],
                  ),
                ),
                const PopupMenuItem(
                  value: 'delete',
                  child: Row(
                    children: [
                      Icon(Icons.delete_outline, size: 18, color: Colors.red),
                      SizedBox(width: 8),
                      Text('Delete', style: TextStyle(color: Colors.red)),
                    ],
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
  static const double _maxAmount = 1000;

  late TimeOfDay _selectedTime;
  late double _amount;
  late Set<int> _selectedDays;
  late final TextEditingController _amountController;
  final FocusNode _amountFocusNode = FocusNode();

  @override
  void initState() {
    super.initState();
    if (widget.schedule != null) {
      final parts = widget.schedule!.time.split(':');
      _selectedTime = TimeOfDay(
        hour: int.tryParse(parts[0]) ?? 8,
        minute: parts.length > 1 ? int.tryParse(parts[1]) ?? 0 : 0,
      );
      _amount = widget.schedule!.amount;
      _selectedDays = widget.schedule!.daysOfWeek.toSet();
    } else if (widget.calculatorSuggestion != null) {
      _selectedTime = const TimeOfDay(hour: 8, minute: 0);
      _amount = _clampAmount(widget.calculatorSuggestion!.recommendedPerFeedingAmount);
      _selectedDays = {0, 1, 2, 3, 4, 5, 6};
    } else {
      _selectedTime = const TimeOfDay(hour: 8, minute: 0);
      _amount = 250.0;
      _selectedDays = {0, 1, 2, 3, 4, 5, 6};
    }

    _amountController = TextEditingController(text: _amount.toInt().toString());
  }

  @override
  void dispose() {
    _amountController.dispose();
    _amountFocusNode.dispose();
    super.dispose();
  }

  double _clampAmount(double value) {
    return value.clamp(_minAmount, _maxAmount).toDouble();
  }

  void _setAmount(double val) {
    final clamped = _clampAmount(val);
    setState(() {
      _amount = clamped;
    });
    if (!_amountFocusNode.hasFocus) {
      _amountController.text = clamped.toInt().toString();
    }
  }

  void _applyCalculatorSuggestion() {
    final suggestion = widget.calculatorSuggestion;
    if (suggestion == null) return;
    final clamped = _clampAmount(suggestion.recommendedPerFeedingAmount);
    setState(() {
      _amount = clamped;
    });
    _amountController.text = clamped.toInt().toString();
  }

  void _save() {
    final timeStr =
        '${_selectedTime.hour.toString().padLeft(2, '0')}:${_selectedTime.minute.toString().padLeft(2, '0')}';
    final schedule = FeedingSchedule(
      id: widget.schedule?.id ?? '',
      deviceId: widget.deviceId,
      time: timeStr,
      amount: _amount,
      durationSeconds: widget.schedule?.durationSeconds ?? 15,
      daysOfWeek: _selectedDays.toList()..sort(),
      isEnabled: widget.schedule?.isEnabled ?? true,
    );
    widget.onSave(schedule);
    Navigator.pop(context);
  }

  @override
  Widget build(BuildContext context) {
    final isDark = Theme.of(context).brightness == Brightness.dark;
    final bgColor = isDark ? const Color(0xFF1E293B) : Colors.white;

    return Container(
      decoration: BoxDecoration(
        color: bgColor,
        borderRadius: const BorderRadius.vertical(top: Radius.circular(24)),
      ),
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom + 20,
        left: 20,
        right: 20,
        top: 20,
      ),
      child: SafeArea(
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              // Header
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: Theme.of(context).colorScheme.primary.withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(10),
                        ),
                        child: Icon(
                          Icons.schedule,
                          color: Theme.of(context).colorScheme.primary,
                          size: 20,
                        ),
                      ),
                      const SizedBox(width: 10),
                      Text(
                        widget.schedule != null ? 'Edit Feeding Schedule' : 'Add Feeding Schedule',
                        style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                      ),
                    ],
                  ),
                  IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: () => Navigator.pop(context),
                  ),
                ],
              ),
              const SizedBox(height: 16),

              // Time Picker Card
              Container(
                decoration: BoxDecoration(
                  color: isDark ? const Color(0xFF0F172A) : const Color(0xFFF1F5F9),
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(color: Colors.white12),
                ),
                child: ListTile(
                  leading: const Icon(Icons.access_time, color: Colors.cyanAccent),
                  title: const Text('Dispense Time', style: TextStyle(fontSize: 13, fontWeight: FontWeight.w500)),
                  subtitle: Text(
                    _selectedTime.format(context),
                    style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 22, color: Colors.cyanAccent),
                  ),
                  trailing: OutlinedButton.icon(
                    onPressed: () async {
                      final time = await showTimePicker(
                        context: context,
                        initialTime: _selectedTime,
                      );
                      if (time != null) {
                        setState(() => _selectedTime = time);
                      }
                    },
                    icon: const Icon(Icons.edit, size: 16),
                    label: const Text('Change'),
                  ),
                ),
              ),
              const SizedBox(height: 16),

              // Amount Card
              Container(
                padding: const EdgeInsets.all(16),
                decoration: BoxDecoration(
                  color: isDark ? const Color(0xFF0F172A) : const Color(0xFFF1F5F9),
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(color: Colors.white12),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        const Text('Dispense Quantity', style: TextStyle(fontWeight: FontWeight.w600, fontSize: 14)),
                        SizedBox(
                          width: 100,
                          child: TextField(
                            controller: _amountController,
                            focusNode: _amountFocusNode,
                            keyboardType: TextInputType.number,
                            textAlign: TextAlign.center,
                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16, color: Colors.cyanAccent),
                            decoration: const InputDecoration(
                              isDense: true,
                              suffixText: 'g',
                              contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 8),
                              border: OutlineInputBorder(),
                            ),
                            onChanged: (text) {
                              final parsed = double.tryParse(text.trim());
                              if (parsed != null) {
                                setState(() {
                                  _amount = _clampAmount(parsed);
                                });
                              }
                            },
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 10),
                    Slider(
                      value: _amount,
                      min: _minAmount,
                      max: _maxAmount,
                      divisions: 99,
                      label: '${_amount.toInt()}g',
                      onChanged: (val) => _setAmount(val),
                    ),
                    // Quick Increments
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceAround,
                      children: [
                        OutlinedButton(
                          style: OutlinedButton.styleFrom(
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                            minimumSize: Size.zero,
                          ),
                          onPressed: () => _setAmount(_amount - 50),
                          child: const Text('-50g', style: TextStyle(fontSize: 11)),
                        ),
                        OutlinedButton(
                          style: OutlinedButton.styleFrom(
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                            minimumSize: Size.zero,
                          ),
                          onPressed: () => _setAmount(_amount - 10),
                          child: const Text('-10g', style: TextStyle(fontSize: 11)),
                        ),
                        OutlinedButton(
                          style: OutlinedButton.styleFrom(
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                            minimumSize: Size.zero,
                          ),
                          onPressed: () => _setAmount(_amount + 10),
                          child: const Text('+10g', style: TextStyle(fontSize: 11)),
                        ),
                        OutlinedButton(
                          style: OutlinedButton.styleFrom(
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                            minimumSize: Size.zero,
                          ),
                          onPressed: () => _setAmount(_amount + 50),
                          child: const Text('+50g', style: TextStyle(fontSize: 11)),
                        ),
                        OutlinedButton(
                          style: OutlinedButton.styleFrom(
                            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                            minimumSize: Size.zero,
                          ),
                          onPressed: () => _setAmount(_amount + 100),
                          child: const Text('+100g', style: TextStyle(fontSize: 11)),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 14),

              // Calculator Suggestion Banner
              if (widget.calculatorSuggestion != null) ...[
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.cyanAccent.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(14),
                    border: Border.all(color: Colors.cyanAccent.withValues(alpha: 0.3)),
                  ),
                  child: Row(
                    children: [
                      const Icon(Icons.auto_awesome, color: Colors.cyanAccent, size: 20),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              'Calculator Recommendation: ${widget.calculatorSuggestion!.recommendedPerFeedingAmount.toInt()}g / feed',
                              style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12),
                            ),
                            Text(
                              'Total daily: ${widget.calculatorSuggestion!.recommendedDailyAmount.toInt()}g (${widget.calculatorSuggestion!.suggestedFeedings} feeds/day)',
                              style: const TextStyle(fontSize: 11, color: Colors.grey),
                            ),
                          ],
                        ),
                      ),
                      TextButton(
                        onPressed: _applyCalculatorSuggestion,
                        child: const Text('Apply', style: TextStyle(fontWeight: FontWeight.bold)),
                      ),
                    ],
                  ),
                ),
                const SizedBox(height: 14),
              ],

              // Repeat Days Selector
              const Text('Active Days of Week', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 14)),
              const SizedBox(height: 8),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  TextButton(
                    onPressed: () => setState(() => _selectedDays = {0, 1, 2, 3, 4, 5, 6}),
                    child: const Text('Everyday', style: TextStyle(fontSize: 12)),
                  ),
                  TextButton(
                    onPressed: () => setState(() => _selectedDays = {1, 2, 3, 4, 5}),
                    child: const Text('Weekdays', style: TextStyle(fontSize: 12)),
                  ),
                  TextButton(
                    onPressed: () => setState(() => _selectedDays = {0, 6}),
                    child: const Text('Weekends', style: TextStyle(fontSize: 12)),
                  ),
                ],
              ),
              Wrap(
                spacing: 6,
                runSpacing: 6,
                children: List.generate(7, (index) {
                  final days = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
                  final isSelected = _selectedDays.contains(index);
                  return FilterChip(
                    label: Text(days[index]),
                    selected: isSelected,
                    selectedColor: Theme.of(context).colorScheme.primary.withValues(alpha: 0.25),
                    checkmarkColor: Theme.of(context).colorScheme.primary,
                    onSelected: (selected) {
                      setState(() {
                        if (selected) {
                          _selectedDays.add(index);
                        } else {
                          if (_selectedDays.length > 1) {
                            _selectedDays.remove(index);
                          }
                        }
                      });
                    },
                  );
                }),
              ),
              const SizedBox(height: 24),

              // Save Button
              FilledButton.icon(
                style: FilledButton.styleFrom(
                  backgroundColor: Theme.of(context).colorScheme.primary,
                  padding: const EdgeInsets.symmetric(vertical: 14),
                  shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(14)),
                ),
                onPressed: _selectedDays.isNotEmpty ? _save : null,
                icon: const Icon(Icons.check_circle_outline),
                label: Text(
                  widget.schedule != null ? 'Update Feeding Schedule' : 'Save Feeding Schedule',
                  style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15),
                ),
              ),
            ],
          ),
        ),
      ),
    );
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
