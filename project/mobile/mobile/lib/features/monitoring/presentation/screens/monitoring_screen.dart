import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:fl_chart/fl_chart.dart';

import '../../../../core/theme/app_theme.dart';
import '../../../../core/models/device.dart';
import '../../../../core/models/sensor_data.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/monitoring_provider.dart';
import '../../../../core/providers/system_health_provider.dart';

class MonitoringScreen extends ConsumerStatefulWidget {
  const MonitoringScreen({super.key});

  @override
  ConsumerState<MonitoringScreen> createState() => _MonitoringScreenState();
}

class _MonitoringScreenState extends ConsumerState<MonitoringScreen> {
  String? _selectedDeviceId;

  @override
  void initState() {
    super.initState();
    Future.microtask(_loadData);
  }

  Future<void> _loadData() async {
    await ref.read(deviceListProvider.notifier).loadDevices();
    if (!mounted) return;
    final devices = ref.read(devicesProvider);
    if (devices.isNotEmpty && _selectedDeviceId == null) {
      _selectedDeviceId = devices.first.id;
      await _loadDeviceData();
    }
  }

  Future<void> _loadDeviceData() async {
    if (!mounted || _selectedDeviceId == null) return;
    final deviceId = _selectedDeviceId!;
    await Future.wait([
      ref.read(sensorDataProvider.notifier).loadSensorData(deviceId),
      ref.read(alertsProvider.notifier).loadAlerts(deviceId),
      ref.read(systemHealthProvider.notifier).loadSystemHealth(deviceId),
    ]);
  }

  @override
  Widget build(BuildContext context) {
    final deviceState = ref.watch(deviceListProvider);
    final sensorState = ref.watch(sensorDataProvider);
    final alertsState = ref.watch(alertsProvider);
    final healthState = ref.watch(systemHealthProvider);
    final sensorData = sensorState.currentData;

    return Scaffold(
      appBar: AppBar(title: const Text('Monitoring')),
      body: RefreshIndicator(
        onRefresh: _loadDeviceData,
        child: SingleChildScrollView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Device selector
              Card(
                child: ListTile(
                  leading: const Icon(Icons.router),
                  title: Text(_getSelectedDeviceName(deviceState.devices)),
                  subtitle: Text(
                    sensorState.lastUpdated != null
                        ? 'Last updated: ${_formatTime(sensorState.lastUpdated!)}'
                        : 'Select a device',
                  ),
                  trailing: const Icon(Icons.arrow_drop_down),
                  onTap:
                      () => _showDeviceSelector(context, deviceState.devices),
                ),
              ),
              const SizedBox(height: 16),

              // ---- System Health Section ----
              _SystemHealthSection(
                healthState: healthState,
                deviceId: _selectedDeviceId,
                onRefresh: () {
                  if (_selectedDeviceId != null) {
                    ref
                        .read(systemHealthProvider.notifier)
                        .loadSystemHealth(_selectedDeviceId!);
                  }
                },
                onRunDiagnostics: () {
                  if (_selectedDeviceId != null) {
                    ref
                        .read(systemHealthProvider.notifier)
                        .triggerDiagnostics(_selectedDeviceId!);
                  }
                },
              ),
              const SizedBox(height: 16),

              Text(
                'Sensor Readings',
                style: Theme.of(
                  context,
                ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 12),

              if (sensorState.isLoading)
                const Center(
                  child: Padding(
                    padding: EdgeInsets.all(32),
                    child: CircularProgressIndicator(),
                  ),
                )
              else if (sensorData == null)
                _buildNoDataCard(context)
              else ...[
                Row(
                  children: [
                    Expanded(
                      child: _SensorCard(
                        icon: Icons.thermostat,
                        label: 'Water Temp',
                        value:
                            '${sensorData.waterTemperature.toStringAsFixed(1)}°C',
                        status:
                            sensorData.temperatureValid
                                ? _getTempStatus(sensorData.waterTemperature)
                                : 'Unavailable',
                        statusColor:
                            sensorData.temperatureValid
                                ? _getTempColor(sensorData.waterTemperature)
                                : Colors.grey,
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _SensorCard(
                        icon: Icons.inventory_2,
                        label: 'Feed Level',
                        value: '${sensorData.feedLevel.toInt()}%',
                        status: _getFeedStatus(sensorData.feedLevel),
                        statusColor: _getFeedColor(sensorData.feedLevel),
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                Row(
                  children: [
                    Expanded(
                      child: _SensorCard(
                        icon: Icons.battery_charging_full,
                        label: 'Battery',
                        value: '${sensorData.batteryLevel.toInt()}%',
                        status:
                            sensorData.isSolarCharging
                                ? 'Charging'
                                : 'Discharging',
                        statusColor: _getBatteryColor(sensorData.batteryLevel),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _SensorCard(
                        icon: Icons.wb_sunny,
                        label: 'Solar',
                        value: '${sensorData.solarVoltage.toStringAsFixed(1)}V',
                        status:
                            sensorData.isSolarCharging ? 'Active' : 'Inactive',
                        statusColor:
                            sensorData.isSolarCharging
                                ? AppTheme.solarActive
                                : Colors.grey,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 12),
                // Connection info
                Card(
                  child: ListTile(
                    leading: Icon(
                      sensorData.connectionType == 'gsm'
                          ? Icons.signal_cellular_alt
                          : Icons.wifi,
                      color: _getSignalColor(sensorData.signalStrength),
                    ),
                    title: Text(sensorData.connectionType.toUpperCase()),
                    subtitle: Text('Signal: ${sensorData.signalStrength}%'),
                    trailing: Icon(
                      Icons.circle,
                      size: 12,
                      color:
                          sensorData.signalStrength > 50
                              ? Colors.green
                              : Colors.orange,
                    ),
                  ),
                ),

                // Q10 Status Card
                const SizedBox(height: 16),
                _Q10StatusCard(temperature: sensorData.waterTemperature),

                // FCR Tracking Card
                const SizedBox(height: 16),
                _FCRTrackingCard(deviceId: _selectedDeviceId!),
              ],

              const SizedBox(height: 24),
              Text(
                'Alerts',
                style: Theme.of(
                  context,
                ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 12),

              if (alertsState.isLoading)
                const Center(child: CircularProgressIndicator())
              else if (alertsState.alerts.isEmpty)
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(24),
                    child: Center(
                      child: Column(
                        children: [
                          Icon(
                            Icons.check_circle,
                            size: 48,
                            color: Colors.green[300],
                          ),
                          const SizedBox(height: 8),
                          Text(
                            'No alerts',
                            style: TextStyle(color: Colors.grey[600]),
                          ),
                        ],
                      ),
                    ),
                  ),
                )
              else
                ...alertsState.alerts
                    .take(5)
                    .map(
                      (alert) => _AlertCard(
                        title: alert.title,
                        message: alert.message,
                        time: _formatTime(alert.createdAt),
                        severity: alert.severity,
                      ),
                    ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildNoDataCard(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Center(
          child: Column(
            children: [
              Icon(Icons.sensors_off, size: 48, color: Colors.grey[400]),
              const SizedBox(height: 8),
              Text(
                'No sensor data available',
                style: TextStyle(color: Colors.grey[600]),
              ),
            ],
          ),
        ),
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
                    _loadDeviceData();
                    Navigator.pop(context);
                  },
                ),
              ),
            ],
          ),
    );
  }

  String _formatTime(DateTime time) {
    final diff = DateTime.now().difference(time);
    if (diff.inMinutes < 1) return 'Just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    return '${diff.inDays}d ago';
  }

  // Status helpers
  String _getTempStatus(double temp) {
    if (temp < 20) return 'Low';
    if (temp > 32) return 'High';
    return 'Normal';
  }

  Color _getTempColor(double temp) {
    if (temp < 20 || temp > 32) return Colors.orange;
    return AppTheme.waterTempNormal;
  }

  String _getFeedStatus(double level) {
    if (level < 20) return 'Critical';
    if (level < 50) return 'Low';
    return 'Good';
  }

  Color _getFeedColor(double level) {
    if (level > 50) return AppTheme.feedLevelHigh;
    if (level > 20) return AppTheme.feedLevelMedium;
    return AppTheme.feedLevelLow;
  }

  Color _getBatteryColor(double level) {
    if (level > 50) return AppTheme.batteryFull;
    if (level > 20) return AppTheme.batteryMedium;
    return AppTheme.batteryLow;
  }

  Color _getSignalColor(int strength) {
    if (strength > 70) return Colors.green;
    if (strength > 40) return Colors.orange;
    return Colors.red;
  }
}

class _SensorCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final String status;
  final Color statusColor;

  const _SensorCard({
    required this.icon,
    required this.label,
    required this.value,
    required this.status,
    required this.statusColor,
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
                Icon(icon, color: statusColor, size: 24),
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
              status,
              style: Theme.of(
                context,
              ).textTheme.bodySmall?.copyWith(color: statusColor),
            ),
          ],
        ),
      ),
    );
  }
}

class _AlertCard extends StatelessWidget {
  final String title;
  final String message;
  final String time;
  final AlertSeverity severity;

  const _AlertCard({
    required this.title,
    required this.message,
    required this.time,
    required this.severity,
  });

  Color get _color {
    switch (severity) {
      case AlertSeverity.info:
        return Colors.blue;
      case AlertSeverity.warning:
        return Colors.orange;
      case AlertSeverity.critical:
        return Colors.red;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: _color.withValues(alpha: 0.2),
          child: Icon(Icons.notifications, color: _color),
        ),
        title: Text(title),
        subtitle: Text(message),
        trailing: Text(time, style: Theme.of(context).textTheme.bodySmall),
      ),
    );
  }
}

/// Q10 Metabolic Status Visualization
class _Q10StatusCard extends StatelessWidget {
  final double temperature;

  const _Q10StatusCard({required this.temperature});

  // Q10 calculation: Q10^((T - Tref) / 10)
  double _calculateQ10Factor(
    double temp, {
    double q10 = 2.2,
    double tRef = 25.0,
  }) {
    return q10 * ((temp - tRef) / 10);
  }

  String _getMetabolicStatus(double temp) {
    if (temp < 18) return 'Low Metabolism';
    if (temp > 32) return 'Thermal Stress';
    if (temp >= 25 && temp <= 30) return 'Optimal';
    return 'Moderate';
  }

  Color _getMetabolicColor(double temp) {
    if (temp < 18 || temp > 32) return Colors.red;
    if (temp >= 25 && temp <= 30) return Colors.green;
    return Colors.orange;
  }

  @override
  Widget build(BuildContext context) {
    final q10Factor = _calculateQ10Factor(temperature);
    final metabolicStatus = _getMetabolicStatus(temperature);
    final statusColor = _getMetabolicColor(temperature);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  Icons.science,
                  color: Theme.of(context).colorScheme.primary,
                ),
                const SizedBox(width: 8),
                Text(
                  'Q10 Metabolic Status',
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),

            // Metabolic gauge
            SizedBox(
              height: 120,
              child: Stack(
                alignment: Alignment.center,
                children: [
                  SizedBox(
                    width: 100,
                    height: 100,
                    child: CircularProgressIndicator(
                      value: (q10Factor.clamp(0, 2) / 2),
                      strokeWidth: 12,
                      backgroundColor: Colors.grey.shade200,
                      valueColor: AlwaysStoppedAnimation(statusColor),
                    ),
                  ),
                  Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        q10Factor.toStringAsFixed(2),
                        style: Theme.of(context).textTheme.headlineSmall
                            ?.copyWith(fontWeight: FontWeight.bold),
                      ),
                      Text(
                        'Factor',
                        style: TextStyle(color: Colors.grey[600], fontSize: 12),
                      ),
                    ],
                  ),
                ],
              ),
            ),

            const SizedBox(height: 16),

            // Status indicators
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: statusColor.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.circle, size: 8, color: statusColor),
                  const SizedBox(width: 8),
                  Text(
                    metabolicStatus,
                    style: TextStyle(
                      color: statusColor,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
            ),

            const SizedBox(height: 16),

            // Factor breakdown
            Row(
              children: [
                Expanded(
                  child: _FactorItem(
                    label: 'Q10 Factor',
                    value: q10Factor.toStringAsFixed(2),
                    icon: Icons.thermostat,
                    color: _getMetabolicColor(temperature),
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

class _FactorItem extends StatelessWidget {
  final String label;
  final String value;
  final IconData icon;
  final Color color;

  const _FactorItem({
    required this.label,
    required this.value,
    required this.icon,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Colors.grey.shade100,
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        children: [
          Icon(icon, color: color, size: 20),
          const SizedBox(height: 4),
          Text(
            value,
            style: TextStyle(fontWeight: FontWeight.bold, color: color),
          ),
          Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])),
        ],
      ),
    );
  }
}

/// FCR Tracking Card with Chart
class _FCRTrackingCard extends ConsumerWidget {
  final String deviceId;

  const _FCRTrackingCard({required this.deviceId});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Mock FCR data - in production, this would come from a provider
    final fcrData = [
      FlSpot(0, 1.8),
      FlSpot(1, 1.7),
      FlSpot(2, 1.6),
      FlSpot(3, 1.5),
      FlSpot(4, 1.4),
      FlSpot(5, 1.3),
      FlSpot(6, 1.25),
    ];

    final currentFCR = fcrData.last.y;
    final targetFCR = 1.2;
    final improvement = ((1.8 - currentFCR) / 1.8 * 100).toStringAsFixed(1);

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  Icons.trending_down,
                  color: Theme.of(context).colorScheme.primary,
                ),
                const SizedBox(width: 8),
                Text(
                  'FCR Tracking',
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const Spacer(),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.green.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    '↓ $improvement%',
                    style: const TextStyle(
                      color: Colors.green,
                      fontWeight: FontWeight.bold,
                      fontSize: 12,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),

            // Current FCR display
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Current FCR',
                        style: TextStyle(color: Colors.grey[600], fontSize: 12),
                      ),
                      Text(
                        currentFCR.toStringAsFixed(2),
                        style: Theme.of(
                          context,
                        ).textTheme.headlineMedium?.copyWith(
                          fontWeight: FontWeight.bold,
                          color:
                              currentFCR <= 1.3
                                  ? Colors.green
                                  : currentFCR <= 1.5
                                  ? Colors.orange
                                  : Colors.red,
                        ),
                      ),
                    ],
                  ),
                ),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Target FCR',
                        style: TextStyle(color: Colors.grey[600], fontSize: 12),
                      ),
                      Text(
                        targetFCR.toStringAsFixed(2),
                        style: Theme.of(
                          context,
                        ).textTheme.headlineMedium?.copyWith(
                          fontWeight: FontWeight.bold,
                          color: Colors.blue,
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),

            const SizedBox(height: 16),

            // FCR trend chart
            SizedBox(
              height: 150,
              child: LineChart(
                LineChartData(
                  gridData: FlGridData(
                    show: true,
                    drawVerticalLine: false,
                    horizontalInterval: 0.2,
                    getDrawingHorizontalLine:
                        (value) =>
                            FlLine(color: Colors.grey.shade200, strokeWidth: 1),
                  ),
                  titlesData: FlTitlesData(
                    leftTitles: AxisTitles(
                      sideTitles: SideTitles(
                        showTitles: true,
                        reservedSize: 30,
                        getTitlesWidget:
                            (value, meta) => Text(
                              value.toStringAsFixed(1),
                              style: TextStyle(
                                fontSize: 10,
                                color: Colors.grey[600],
                              ),
                            ),
                      ),
                    ),
                    bottomTitles: AxisTitles(
                      sideTitles: SideTitles(
                        showTitles: true,
                        getTitlesWidget:
                            (value, meta) => Text(
                              'W${value.toInt() + 1}',
                              style: TextStyle(
                                fontSize: 10,
                                color: Colors.grey[600],
                              ),
                            ),
                      ),
                    ),
                    topTitles: const AxisTitles(
                      sideTitles: SideTitles(showTitles: false),
                    ),
                    rightTitles: const AxisTitles(
                      sideTitles: SideTitles(showTitles: false),
                    ),
                  ),
                  borderData: FlBorderData(show: false),
                  minX: 0,
                  maxX: 6,
                  minY: 1.0,
                  maxY: 2.0,
                  lineBarsData: [
                    // Target line
                    LineChartBarData(
                      spots: [const FlSpot(0, 1.2), const FlSpot(6, 1.2)],
                      isCurved: false,
                      color: Colors.blue.withValues(alpha: 0.5),
                      barWidth: 2,
                      dotData: const FlDotData(show: false),
                      dashArray: [5, 5],
                    ),
                    // Actual FCR
                    LineChartBarData(
                      spots: fcrData,
                      isCurved: true,
                      color: Colors.green,
                      barWidth: 3,
                      dotData: FlDotData(
                        show: true,
                        getDotPainter:
                            (spot, percent, barData, index) =>
                                FlDotCirclePainter(
                                  radius: 4,
                                  color: Colors.green,
                                  strokeWidth: 2,
                                  strokeColor: Colors.white,
                                ),
                      ),
                      belowBarData: BarAreaData(
                        show: true,
                        color: Colors.green.withValues(alpha: 0.1),
                      ),
                    ),
                  ],
                ),
              ),
            ),

            const SizedBox(height: 12),

            // Legend
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                _LegendItem(color: Colors.green, label: 'Actual FCR'),
                const SizedBox(width: 16),
                _LegendItem(
                  color: Colors.blue,
                  label: 'Target (1.2)',
                  dashed: true,
                ),
              ],
            ),

            const SizedBox(height: 12),

            // Recommendation
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: Colors.blue.withValues(alpha: 0.1),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                children: [
                  const Icon(
                    Icons.lightbulb_outline,
                    size: 16,
                    color: Colors.blue,
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      currentFCR <= targetFCR
                          ? 'Excellent! FCR is at optimal level.'
                          : 'Tip: Optimize feeding times during peak metabolic hours.',
                      style: const TextStyle(fontSize: 12, color: Colors.blue),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _LegendItem extends StatelessWidget {
  final Color color;
  final String label;
  final bool dashed;

  const _LegendItem({
    required this.color,
    required this.label,
    this.dashed = false,
  });

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 16,
          height: 3,
          decoration: BoxDecoration(
            color: dashed ? Colors.transparent : color,
            border:
                dashed
                    ? Border(
                      bottom: BorderSide(
                        color: color,
                        width: 2,
                        style: BorderStyle.solid,
                      ),
                    )
                    : null,
          ),
        ),
        const SizedBox(width: 4),
        Text(label, style: TextStyle(fontSize: 11, color: Colors.grey[600])),
      ],
    );
  }
}

// =============================================================================
// System Health Section
// =============================================================================

class _SystemHealthSection extends StatefulWidget {
  final SystemHealthState healthState;
  final String? deviceId;
  final VoidCallback onRefresh;
  final VoidCallback onRunDiagnostics;

  const _SystemHealthSection({
    required this.healthState,
    required this.deviceId,
    required this.onRefresh,
    required this.onRunDiagnostics,
  });

  @override
  State<_SystemHealthSection> createState() => _SystemHealthSectionState();
}

class _SystemHealthSectionState extends State<_SystemHealthSection> {
  bool _isExpanded = true;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final h = widget.healthState;

    return Card(
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          // Header
          InkWell(
            onTap: () => setState(() => _isExpanded = !_isExpanded),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              child: Row(
                children: [
                  Icon(
                    Icons.health_and_safety,
                    color:
                        h.allComponentsHealthy
                            ? AppTheme.deviceOnline
                            : (h.errorCount > 0
                                ? AppTheme.feedLevelLow
                                : Colors.grey),
                    size: 24,
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'System Health',
                          style: theme.textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                          ),
                        ),
                        if (h.components.isNotEmpty)
                          Text(
                            '${h.okCount}/${h.components.length} components OK',
                            style: theme.textTheme.bodySmall?.copyWith(
                              color:
                                  h.allComponentsHealthy
                                      ? AppTheme.deviceOnline
                                      : AppTheme.feedLevelLow,
                            ),
                          )
                        else if (h.isLoading)
                          Text('Loading...', style: theme.textTheme.bodySmall)
                        else
                          Text(
                            h.message ?? 'No diagnostics yet',
                            style: theme.textTheme.bodySmall,
                          ),
                      ],
                    ),
                  ),
                  Icon(
                    _isExpanded
                        ? Icons.keyboard_arrow_up
                        : Icons.keyboard_arrow_down,
                  ),
                ],
              ),
            ),
          ),

          if (_isExpanded) ...[
            const Divider(height: 1),

            if (h.isLoading)
              const Padding(
                padding: EdgeInsets.all(24),
                child: CircularProgressIndicator(),
              )
            else if (h.error != null)
              Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  children: [
                    Icon(
                      Icons.error_outline,
                      color: AppTheme.feedLevelLow,
                      size: 32,
                    ),
                    const SizedBox(height: 8),
                    Text(h.error!, textAlign: TextAlign.center),
                    const SizedBox(height: 12),
                    TextButton.icon(
                      onPressed: widget.onRefresh,
                      icon: const Icon(Icons.refresh),
                      label: const Text('Retry'),
                    ),
                  ],
                ),
              )
            else ...[
              // ---- Pipeline Connectivity Chain ----
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
                child: Text(
                  'Pipeline Connectivity',
                  style: theme.textTheme.labelLarge,
                ),
              ),
              _PipelineChain(pipeline: h.pipeline),

              const Divider(height: 1),

              // ---- Hardware Components ----
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
                child: Text(
                  'Hardware Components',
                  style: theme.textTheme.labelLarge,
                ),
              ),
              if (h.components.isEmpty)
                Padding(
                  padding: const EdgeInsets.all(16),
                  child: Text(
                    'Awaiting diagnostics report from device...',
                    style: theme.textTheme.bodySmall,
                  ),
                )
              else
                ...h.components.map((c) => _ComponentTile(component: c)),

              // ---- ESP32-CAM Independence ----
              if (h.canWorkWithoutCam == true)
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 4, 16, 4),
                  child: Row(
                    children: [
                      Icon(
                        Icons.info_outline,
                        size: 16,
                        color: Colors.blue[400],
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          'System works without ESP32-CAM — camera is optional for visual feeding verification only.',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: Colors.blue[600],
                            fontStyle: FontStyle.italic,
                          ),
                        ),
                      ),
                    ],
                  ),
                ),

              const Divider(height: 1),

              // ---- Backend Services ----
              if (h.backendHealth.isNotEmpty) ...[
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
                  child: Text(
                    'Backend Services',
                    style: theme.textTheme.labelLarge,
                  ),
                ),
                ...h.backendHealth.map(
                  (b) => ListTile(
                    dense: true,
                    leading: Icon(
                      b.isOk ? Icons.check_circle : Icons.error,
                      color:
                          b.isOk
                              ? AppTheme.deviceOnline
                              : AppTheme.feedLevelLow,
                      size: 20,
                    ),
                    title: Text(
                      _capitalise(b.name),
                      style: const TextStyle(fontSize: 14),
                    ),
                    subtitle: Text(
                      b.message,
                      style: const TextStyle(fontSize: 12),
                    ),
                  ),
                ),
                const Divider(height: 1),
              ],

              // ---- Actions ----
              Padding(
                padding: const EdgeInsets.all(12),
                child: Row(
                  children: [
                    Expanded(
                      child: OutlinedButton.icon(
                        onPressed: widget.onRefresh,
                        icon: const Icon(Icons.refresh, size: 18),
                        label: const Text('Refresh'),
                      ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: FilledButton.icon(
                        onPressed: widget.onRunDiagnostics,
                        icon: const Icon(Icons.play_arrow, size: 18),
                        label: const Text('Run Diagnostics'),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ],
        ],
      ),
    );
  }

  static String _capitalise(String s) {
    if (s.isEmpty) return s;
    return s[0].toUpperCase() + s.substring(1);
  }
}

// =============================================================================
// Hardware Component Tile
// =============================================================================

class _ComponentTile extends StatelessWidget {
  final ComponentStatus component;

  const _ComponentTile({required this.component});

  @override
  Widget build(BuildContext context) {
    IconData icon;
    Color color;

    switch (component.status) {
      case 'ok':
        icon = Icons.check_circle;
        color = AppTheme.deviceOnline;
        break;
      case 'error':
        icon = Icons.cancel;
        color = AppTheme.feedLevelLow;
        break;
      case 'neutral':
        icon = Icons.remove_circle_outline;
        color = Colors.grey;
        break;
      case 'skipped':
        icon = Icons.block;
        color = Colors.grey[400]!;
        break;
      default:
        icon = Icons.help_outline;
        color = Colors.grey;
    }

    return ListTile(
      dense: true,
      leading: Icon(icon, color: color, size: 22),
      title: Text(
        component.name,
        style: const TextStyle(fontSize: 14, fontWeight: FontWeight.w500),
      ),
      subtitle: Text(
        component.message,
        style: TextStyle(fontSize: 12, color: Colors.grey[600]),
      ),
    );
  }
}

// =============================================================================
// Pipeline Connectivity Chain
// =============================================================================

class _PipelineChain extends StatelessWidget {
  final PipelineHealth pipeline;

  const _PipelineChain({required this.pipeline});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          _PipelineNode(
            label: 'MCU',
            icon: Icons.memory,
            isConnected: pipeline.mcuToMqtt,
          ),
          _PipelineConnector(isConnected: pipeline.mcuToMqtt),
          _PipelineNode(
            label: 'MQTT',
            icon: Icons.cloud_queue,
            isConnected: pipeline.mqttToBackend,
          ),
          _PipelineConnector(isConnected: pipeline.mqttToBackend),
          _PipelineNode(
            label: 'Backend',
            icon: Icons.dns,
            isConnected: pipeline.backendToApp,
          ),
          _PipelineConnector(isConnected: pipeline.appToBackend),
          _PipelineNode(
            label: 'App',
            icon: Icons.phone_android,
            isConnected: pipeline.appToBackend,
          ),
        ],
      ),
    );
  }
}

class _PipelineNode extends StatelessWidget {
  final String label;
  final IconData icon;
  final bool? isConnected;

  const _PipelineNode({
    required this.label,
    required this.icon,
    required this.isConnected,
  });

  @override
  Widget build(BuildContext context) {
    Color color;
    if (isConnected == true) {
      color = AppTheme.deviceOnline;
    } else if (isConnected == false) {
      color = AppTheme.feedLevelLow;
    } else {
      color = Colors.grey;
    }

    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        Container(
          width: 40,
          height: 40,
          decoration: BoxDecoration(
            color: color.withValues(alpha: 0.1),
            shape: BoxShape.circle,
            border: Border.all(color: color, width: 2),
          ),
          child: Icon(icon, size: 20, color: color),
        ),
        const SizedBox(height: 4),
        Text(
          label,
          style: TextStyle(
            fontSize: 10,
            fontWeight: FontWeight.w600,
            color: color,
          ),
        ),
      ],
    );
  }
}

class _PipelineConnector extends StatelessWidget {
  final bool? isConnected;

  const _PipelineConnector({required this.isConnected});

  @override
  Widget build(BuildContext context) {
    Color color;
    if (isConnected == true) {
      color = AppTheme.deviceOnline;
    } else if (isConnected == false) {
      color = AppTheme.feedLevelLow;
    } else {
      color = Colors.grey[300]!;
    }

    return Padding(
      padding: const EdgeInsets.only(bottom: 16),
      child: SizedBox(
        width: 24,
        child: Row(
          children: [
            Expanded(child: Container(height: 2, color: color)),
            Icon(
              isConnected == true ? Icons.arrow_forward_ios : Icons.remove,
              size: 8,
              color: color,
            ),
          ],
        ),
      ),
    );
  }
}
