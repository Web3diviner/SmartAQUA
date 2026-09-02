import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/models/device.dart';
import '../../../../core/models/feeding.dart';
import '../../../../core/models/sensor_data.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/feeding_provider.dart';
import '../../../../core/providers/monitoring_provider.dart';
import '../../../../core/providers/realtime_provider.dart';

class DeviceDetailScreen extends ConsumerStatefulWidget {
  final String deviceId;

  const DeviceDetailScreen({super.key, required this.deviceId});

  @override
  ConsumerState<DeviceDetailScreen> createState() => _DeviceDetailScreenState();
}

class _DeviceDetailScreenState extends ConsumerState<DeviceDetailScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(_loadData);
  }

  Future<void> _loadData() async {
    await Future.wait([
      ref.read(selectedDeviceProvider.notifier).loadDevice(widget.deviceId),
      ref.read(sensorDataProvider.notifier).loadSensorData(widget.deviceId),
      ref
          .read(feedingHistoryProvider.notifier)
          .loadHistory(widget.deviceId, refresh: true),
    ]);
    if (!mounted) return;

    final realtime = ref.read(realtimeProvider.notifier);
    final connected = await realtime.connect();
    if (!mounted) return;
    if (connected) {
      realtime.subscribeToDevice(widget.deviceId);
    }
  }

  @override
  void dispose() {
    ref.read(realtimeProvider.notifier).unsubscribeFromDevice(widget.deviceId);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final deviceState = ref.watch(selectedDeviceProvider);
    final sensorState = ref.watch(sensorDataProvider);
    final feedingState = ref.watch(feedingHistoryProvider);
    final device = deviceState.device;

    return Scaffold(
      appBar: AppBar(
        title: Text(device?.name ?? 'Device Details'),
        actions: [
          IconButton(
            icon: const Icon(Icons.videocam),
            onPressed: () => context.go('/video'),
            tooltip: 'Video Clips',
          ),
          IconButton(
            icon: const Icon(Icons.settings),
            onPressed: () => _showDeviceSettings(context),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _loadData,
        child:
            deviceState.isLoading
                ? const Center(child: CircularProgressIndicator())
                : device == null
                ? _buildErrorState()
                : SingleChildScrollView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // Device Info Card
                      _buildDeviceInfoCard(context, device),
                      const SizedBox(height: 16),

                      // Status Cards
                      _buildStatusCards(
                        context,
                        device,
                        sensorState.currentData,
                      ),
                      const SizedBox(height: 24),

                      // Quick Actions
                      _buildQuickActions(context, device),
                      const SizedBox(height: 24),

                      // Recent Feeding
                      _buildRecentFeeding(context, feedingState),
                    ],
                  ),
                ),
      ),
    );
  }

  Widget _buildErrorState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.error_outline, size: 64, color: Colors.grey[400]),
          const SizedBox(height: 16),
          Text('Device not found', style: TextStyle(color: Colors.grey[600])),
          const SizedBox(height: 16),
          ElevatedButton(onPressed: _loadData, child: const Text('Retry')),
        ],
      ),
    );
  }

  Widget _buildDeviceInfoCard(BuildContext context, Device device) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  Icons.router,
                  size: 48,
                  color:
                      device.isOnline
                          ? AppTheme.deviceOnline
                          : AppTheme.deviceOffline,
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        device.name,
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      Text(
                        device.serialNumber,
                        style: Theme.of(
                          context,
                        ).textTheme.bodyMedium?.copyWith(color: Colors.grey),
                      ),
                    ],
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 12,
                    vertical: 6,
                  ),
                  decoration: BoxDecoration(
                    color: (device.isOnline
                            ? AppTheme.deviceOnline
                            : AppTheme.deviceOffline)
                        .withValues(alpha: 0.2),
                    borderRadius: BorderRadius.circular(16),
                  ),
                  child: Text(
                    device.isOnline ? 'Online' : 'Offline',
                    style: TextStyle(
                      color:
                          device.isOnline
                              ? AppTheme.deviceOnline
                              : AppTheme.deviceOffline,
                      fontWeight: FontWeight.w500,
                    ),
                  ),
                ),
              ],
            ),
            if (device.lastSeen != null) ...[
              const SizedBox(height: 12),
              Text(
                'Last seen: ${_formatLastSeen(device.lastSeen!)}',
                style: TextStyle(color: Colors.grey[600], fontSize: 12),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildStatusCards(
    BuildContext context,
    Device device,
    SensorData? sensorData,
  ) {
    final baseStatus = device.status;
    final batteryLevel = sensorData?.batteryLevel ?? baseStatus.batteryLevel;
    final feedLevel = sensorData?.feedLevel ?? baseStatus.feedLevel;
    final waterTemperature =
        sensorData?.temperatureValid == true
            ? sensorData!.waterTemperature
            : baseStatus.waterTemperature;
    final signalStrength =
        sensorData?.signalStrength ?? baseStatus.signalStrength;
    final solarVoltage = sensorData?.solarVoltage ?? baseStatus.solarVoltage;
    final isSolarCharging =
        sensorData?.isSolarCharging ?? baseStatus.isSolarCharging;
    final connectionType =
        sensorData?.connectionType ?? baseStatus.connectionType;
    final status = DeviceStatus(
      batteryLevel: batteryLevel,
      feedLevel: feedLevel,
      waterTemperature: waterTemperature,
      signalStrength: signalStrength,
      isSolarCharging: isSolarCharging,
      solarVoltage: solarVoltage,
      connectionType: connectionType,
    );

    return Column(
      children: [
        Row(
          children: [
            Expanded(
              child: _StatusCard(
                icon: Icons.battery_charging_full,
                label: 'Battery',
                value: '${batteryLevel.toInt()}%',
                subtitle: isSolarCharging ? 'Solar charging' : 'Discharging',
                color: _getBatteryColor(batteryLevel),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: _StatusCard(
                icon: Icons.inventory_2,
                label: 'Feed Level',
                value: '${feedLevel.toInt()}%',
                subtitle: _getFeedLevelText(feedLevel),
                color: _getFeedLevelColor(feedLevel),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: _StatusCard(
                icon: Icons.thermostat,
                label: 'Water Temp',
                value: '${status.waterTemperature.toStringAsFixed(1)}°C',
                subtitle: _getTempStatus(waterTemperature),
                color: _getTempColor(waterTemperature),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: _StatusCard(
                icon:
                    status.connectionType == 'gsm'
                        ? Icons.signal_cellular_alt
                        : Icons.wifi,
                label: 'Signal',
                value: '${status.signalStrength}%',
                subtitle: status.connectionType.toUpperCase(),
                color: _getSignalColor(status.signalStrength),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),
        _StatusCard(
          icon: Icons.wb_sunny,
          label: 'Solar',
          value: '${status.solarVoltage.toStringAsFixed(1)}V',
          subtitle: status.isSolarCharging ? 'Active' : 'Inactive',
          color: status.isSolarCharging ? AppTheme.solarActive : Colors.grey,
        ),
      ],
    );
  }

  Widget _buildQuickActions(BuildContext context, Device device) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Quick Actions',
          style: Theme.of(
            context,
          ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
        ),
        const SizedBox(height: 12),
        Row(
          children: [
            Expanded(
              child: FilledButton.icon(
                onPressed:
                    device.isOnline
                        ? () => context.go('/feeding/manual')
                        : null,
                icon: const Icon(Icons.restaurant),
                label: const Text('Feed Now'),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () => context.go('/feeding'),
                icon: const Icon(Icons.schedule),
                label: const Text('Schedules'),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        Row(
          children: [
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () => context.go('/monitoring'),
                icon: const Icon(Icons.monitor_heart),
                label: const Text('Monitor'),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () => context.go('/video'),
                icon: const Icon(Icons.videocam),
                label: const Text('Videos'),
              ),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildRecentFeeding(
    BuildContext context,
    FeedingHistoryState feedingState,
  ) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              'Recent Feeding',
              style: Theme.of(
                context,
              ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
            ),
            TextButton(
              onPressed: () => context.go('/feeding/history'),
              child: const Text('View All'),
            ),
          ],
        ),
        const SizedBox(height: 8),
        if (feedingState.isLoading)
          const Center(child: CircularProgressIndicator())
        else if (feedingState.events.isEmpty)
          Card(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Center(
                child: Text(
                  'No feeding history',
                  style: TextStyle(color: Colors.grey[600]),
                ),
              ),
            ),
          )
        else
          ...feedingState.events
              .take(3)
              .map(
                (event) => _FeedingHistoryItem(
                  time: DateFormat('h:mm a').format(event.scheduledAt),
                  amount: '${event.amount.toInt()}g',
                  type: event.type == 'manual' ? 'Manual' : 'Scheduled',
                  status: _feedingStatusLabel(event.status),
                  isPending: event.status == FeedingEventStatus.pending,
                ),
              ),
      ],
    );
  }

  String _feedingStatusLabel(FeedingEventStatus status) {
    switch (status) {
      case FeedingEventStatus.completed:
        return 'completed';
      case FeedingEventStatus.failed:
        return 'failed';
      case FeedingEventStatus.pending:
        return 'pending';
      case FeedingEventStatus.cancelled:
        return 'cancelled';
    }
  }

  void _showDeviceSettings(BuildContext context) {
    showModalBottomSheet(
      context: context,
      builder:
          (ctx) => Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                ListTile(
                  leading: const Icon(Icons.edit),
                  title: const Text('Rename Device'),
                  onTap: () {
                    Navigator.pop(ctx);
                    _showRenameDialog(context);
                  },
                ),
                ListTile(
                  leading: const Icon(Icons.notifications),
                  title: const Text('Notification Settings'),
                  onTap: () {
                    Navigator.pop(ctx);
                    context.go('/settings');
                  },
                ),
                ListTile(
                  leading: const Icon(Icons.tune),
                  title: const Text('Threshold Settings'),
                  onTap: () {
                    Navigator.pop(ctx);
                  },
                ),
                ListTile(
                  leading: const Icon(Icons.link_off, color: Colors.red),
                  title: const Text(
                    'Unbind Device',
                    style: TextStyle(color: Colors.red),
                  ),
                  onTap: () {
                    Navigator.pop(ctx);
                    _showUnbindConfirmation(context);
                  },
                ),
              ],
            ),
          ),
    );
  }

  void _showRenameDialog(BuildContext context) {
    final controller = TextEditingController();
    showDialog(
      context: context,
      builder:
          (ctx) => AlertDialog(
            title: const Text('Rename Device'),
            content: TextField(
              controller: controller,
              decoration: const InputDecoration(
                labelText: 'Device Name',
                border: OutlineInputBorder(),
              ),
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text('Cancel'),
              ),
              FilledButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text('Save'),
              ),
            ],
          ),
    );
  }

  void _showUnbindConfirmation(BuildContext context) {
    showDialog(
      context: context,
      builder:
          (ctx) => AlertDialog(
            title: const Text('Unbind Device'),
            content: const Text(
              'Are you sure you want to unbind this device? You will need to pair it again to use it.',
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text('Cancel'),
              ),
              TextButton(
                onPressed: () async {
                  Navigator.pop(ctx);
                  final success = await ref
                      .read(deviceListProvider.notifier)
                      .unbindDevice(widget.deviceId);
                  if (success && context.mounted) {
                    context.go('/devices');
                  }
                },
                child: const Text(
                  'Unbind',
                  style: TextStyle(color: Colors.red),
                ),
              ),
            ],
          ),
    );
  }

  String _formatLastSeen(DateTime lastSeen) {
    final diff = DateTime.now().difference(lastSeen);
    if (diff.inMinutes < 1) return 'Just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes} minutes ago';
    if (diff.inHours < 24) return '${diff.inHours} hours ago';
    return DateFormat('MMM d, h:mm a').format(lastSeen);
  }

  Color _getBatteryColor(double level) {
    if (level > 50) return AppTheme.batteryFull;
    if (level > 20) return AppTheme.batteryMedium;
    return AppTheme.batteryLow;
  }

  Color _getFeedLevelColor(double level) {
    if (level > 50) return AppTheme.feedLevelHigh;
    if (level > 20) return AppTheme.feedLevelMedium;
    return AppTheme.feedLevelLow;
  }

  String _getFeedLevelText(double level) {
    if (level > 50) return 'Good';
    if (level > 20) return 'Low';
    return 'Critical';
  }

  Color _getTempColor(double temp) {
    if (temp < 20 || temp > 32) return Colors.orange;
    return AppTheme.waterTempNormal;
  }

  String _getTempStatus(double temp) {
    if (temp < 20) return 'Low';
    if (temp > 32) return 'High';
    return 'Normal';
  }

  Color _getSignalColor(int strength) {
    if (strength > 70) return Colors.green;
    if (strength > 40) return Colors.orange;
    return Colors.red;
  }
}

class _StatusCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final String subtitle;
  final Color color;

  const _StatusCard({
    required this.icon,
    required this.label,
    required this.value,
    required this.subtitle,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, color: color, size: 24),
                const SizedBox(width: 8),
                Text(
                  label,
                  style: Theme.of(
                    context,
                  ).textTheme.bodyMedium?.copyWith(color: Colors.grey),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              value,
              style: Theme.of(
                context,
              ).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.bold),
            ),
            Text(
              subtitle,
              style: Theme.of(
                context,
              ).textTheme.bodySmall?.copyWith(color: Colors.grey),
            ),
          ],
        ),
      ),
    );
  }
}

class _FeedingHistoryItem extends StatelessWidget {
  final String time;
  final String amount;
  final String type;
  final String status;
  final bool isPending;

  const _FeedingHistoryItem({
    required this.time,
    required this.amount,
    required this.type,
    required this.status,
    this.isPending = false,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor:
              isPending
                  ? Colors.orange.withValues(alpha: 0.2)
                  : AppTheme.feedLevelHigh.withValues(alpha: 0.2),
          child: Icon(
            isPending ? Icons.schedule : Icons.check,
            color: isPending ? Colors.orange : AppTheme.feedLevelHigh,
          ),
        ),
        title: Text('$amount - $type'),
        subtitle: Text(time),
        trailing: Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          decoration: BoxDecoration(
            color:
                isPending
                    ? Colors.orange.withValues(alpha: 0.2)
                    : AppTheme.feedLevelHigh.withValues(alpha: 0.2),
            borderRadius: BorderRadius.circular(8),
          ),
          child: Text(
            status,
            style: TextStyle(
              color: isPending ? Colors.orange : AppTheme.feedLevelHigh,
              fontSize: 12,
            ),
          ),
        ),
      ),
    );
  }
}
