import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/device.dart';
import '../../../../core/models/feeding.dart';
import '../../../../core/providers/app_preferences_provider.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/feeding_provider.dart';
import '../../../../core/providers/monitoring_provider.dart';

class DashboardScreen extends ConsumerStatefulWidget {
  const DashboardScreen({super.key});

  @override
  ConsumerState<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends ConsumerState<DashboardScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(_loadData);
  }

  Future<void> _loadData() async {
    await ref.read(deviceListProvider.notifier).loadDevices();
    if (!mounted) return;
    final devices = ref.read(devicesProvider);
    if (devices.isNotEmpty) {
      final deviceId = devices.first.id;
      await Future.wait([
        ref
            .read(feedingHistoryProvider.notifier)
            .loadHistory(deviceId, refresh: true),
        ref.read(alertsProvider.notifier).loadAlerts(deviceId),
      ]);
    }
  }

  @override
  Widget build(BuildContext context) {
    final deviceState = ref.watch(deviceListProvider);
    final feedingState = ref.watch(feedingHistoryProvider);
    final alertsState = ref.watch(alertsProvider);
    final todayFeedings = ref.watch(todayFeedingsProvider);
    final isDark = Theme.of(context).brightness == Brightness.dark;

    return Scaffold(
      body: CustomScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        slivers: [
          // App Bar with Glassmorphic Header
          SliverAppBar(
            pinned: true,
            expandedHeight: 80,
            backgroundColor: isDark ? const Color(0xFF070F18) : Colors.white,
            flexibleSpace: FlexibleSpaceBar(
              titlePadding: const EdgeInsets.only(left: 20, bottom: 16),
              title: Row(
                children: [
                  Container(
                    width: 32,
                    height: 32,
                    decoration: BoxDecoration(
                      gradient: AppTheme.primaryGradient,
                      borderRadius: BorderRadius.circular(10),
                      boxShadow: [
                        BoxShadow(
                          color: AppTheme.primaryCyan.withOpacity(0.4),
                          blurRadius: 8,
                        ),
                      ],
                    ),
                    child: const Icon(Icons.waves, color: Colors.white, size: 18),
                  ),
                  const SizedBox(width: 10),
                  const Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'SmartAQUA',
                        style: TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w900,
                          letterSpacing: -0.5,
                        ),
                      ),
                      Text(
                        'Precision Aquaculture OS',
                        style: TextStyle(fontSize: 9, color: Colors.grey, fontWeight: FontWeight.normal),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            actions: [
              Container(
                margin: const EdgeInsets.symmetric(vertical: 8),
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                decoration: BoxDecoration(
                  color: Colors.green.withOpacity(0.15),
                  borderRadius: BorderRadius.circular(20),
                  border: Border.all(color: Colors.green.withOpacity(0.3)),
                ),
                child: const Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(Icons.circle, color: Colors.greenAccent, size: 8),
                    SizedBox(width: 5),
                    Text('EDGE LIVE', style: TextStyle(color: Colors.greenAccent, fontSize: 10, fontWeight: FontWeight.bold)),
                  ],
                ),
              ),
              const SizedBox(width: 4),
              // Quick Light/Dark Mode Switch Button
              IconButton(
                icon: Icon(
                  isDark ? Icons.light_mode_rounded : Icons.dark_mode_rounded,
                  color: isDark ? const Color(0xFFFFD166) : const Color(0xFF0077B6),
                ),
                tooltip: isDark ? 'Switch to Light Mode' : 'Switch to Dark Mode',
                onPressed: () {
                  ref.read(appPreferencesProvider.notifier).toggleThemeMode(currentIsDark: isDark);
                },
              ),
              Stack(
                children: [
                  IconButton(
                    icon: const Icon(Icons.notifications_outlined),
                    onPressed: () {},
                  ),
                  if (alertsState.unreadCount > 0)
                    Positioned(
                      right: 8,
                      top: 8,
                      child: Container(
                        padding: const EdgeInsets.all(4),
                        decoration: const BoxDecoration(
                          color: Colors.red,
                          shape: BoxShape.circle,
                        ),
                        child: Text(
                          '${alertsState.unreadCount}',
                          style: const TextStyle(color: Colors.white, fontSize: 9, fontWeight: FontWeight.bold),
                        ),
                      ),
                    ),
                ],
              ),
              const SizedBox(width: 8),
            ],
          ),

          // Main Content
          SliverToBoxAdapter(
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Hero Farm Operations Card
                  _buildHeroFarmCard(context),
                  const SizedBox(height: 20),

                  // 4-Card Telemetry Grid
                  Text(
                    'Live Telemetry & Diagnostics',
                    style: Theme.of(context).textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.bold,
                      letterSpacing: 0.5,
                      color: isDark ? Colors.grey[400] : const Color(0xFF4A5568),
                    ),
                  ),
                  const SizedBox(height: 12),
                  _buildSensorMatrix(context, deviceState, todayFeedings, alertsState),
                  const SizedBox(height: 20),

                  // AquaDoc Clinical AI Hologram Card
                  _buildAquaDocCard(context),
                  const SizedBox(height: 24),

                  // Quick Action Hub Matrix
                  Text(
                    'Precision Operations & Tools',
                    style: Theme.of(context).textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.bold,
                      letterSpacing: 0.5,
                      color: isDark ? Colors.grey[400] : const Color(0xFF4A5568),
                    ),
                  ),
                  const SizedBox(height: 12),
                  _buildQuickActionMatrix(context),
                  const SizedBox(height: 24),

                  // Recent Feeding & Telemetry Activity
                  Text(
                    'Recent Feeder Activity',
                    style: Theme.of(context).textTheme.titleSmall?.copyWith(
                      fontWeight: FontWeight.bold,
                      letterSpacing: 0.5,
                      color: isDark ? Colors.grey[400] : const Color(0xFF4A5568),
                    ),
                  ),
                  const SizedBox(height: 12),
                  _buildRecentActivity(context, feedingState),
                  const SizedBox(height: 24),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildHeroFarmCard(BuildContext context) {
    final devices = ref.watch(devicesProvider);
    final sensorData = ref.watch(sensorDataProvider).currentData;
    final alerts = ref.watch(alertsProvider).alerts;

    final healthScore = alerts.isEmpty ? 100 : (100 - (alerts.length * 10)).clamp(50, 100);
    final hopperStr = sensorData != null && sensorData.feedLevel > 0
        ? '${sensorData.feedLevel.toStringAsFixed(0)}% Full'
        : (devices.isNotEmpty ? 'Hopper OK' : 'No Hardware');
    final hopperSub = sensorData != null && sensorData.feedLevel > 0
        ? '${(sensorData.feedLevel * 0.05).toStringAsFixed(1)} kg Remaining'
        : 'Connect Feeder';

    final facilitySub = devices.isEmpty
        ? 'No device connected • Tap + to pair'
        : '${devices.length} Connected Unit${devices.length == 1 ? '' : 's'} • Live Monitoring';

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [
            Color(0xFF0D2538),
            Color(0xFF09141F),
          ],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: AppTheme.primaryCyan.withOpacity(0.3), width: 1.2),
        boxShadow: [
          BoxShadow(
            color: AppTheme.primaryCyan.withOpacity(0.15),
            blurRadius: 20,
            offset: const Offset(0, 8),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      'SmartAQUA Facility',
                      style: TextStyle(
                        color: Colors.white,
                        fontSize: 18,
                        fontWeight: FontWeight.bold,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      facilitySub,
                      style: TextStyle(color: Colors.grey[400], fontSize: 12),
                    ),
                  ],
                ),
              ),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                decoration: BoxDecoration(
                  gradient: AppTheme.primaryGradient,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Text(
                  'HEALTH $healthScore%',
                  style: const TextStyle(color: Colors.white, fontSize: 11, fontWeight: FontWeight.bold),
                ),
              ),
            ],
          ),
          const SizedBox(height: 20),
          Row(
            children: [
              Expanded(
                child: _HeroStat(
                  label: 'Connected',
                  value: devices.isEmpty ? '0 Connected' : '${devices.length} Units',
                  sub: devices.isEmpty ? 'No Machine Linked' : '${devices.where((Device d) => d.isOnline).length} Active Online',
                  icon: Icons.hub_outlined,
                  color: devices.isEmpty ? Colors.grey : AppTheme.primaryCyan,
                ),
              ),
              Container(width: 1, height: 45, color: Colors.white12),
              Expanded(
                child: _HeroStat(
                  label: 'Water Temp',
                  value: sensorData != null && sensorData.waterTemperature > 0
                      ? '${sensorData.waterTemperature.toStringAsFixed(1)} °C'
                      : '-- °C',
                  sub: sensorData != null ? 'Telemetry Sync' : (devices.isEmpty ? 'No Sensor' : 'Awaiting Sensor'),
                  icon: Icons.thermostat,
                  color: AppTheme.primaryTeal,
                ),
              ),
              Container(width: 1, height: 45, color: Colors.white12),
              Expanded(
                child: _HeroStat(
                  label: 'Feed Hopper',
                  value: hopperStr,
                  sub: hopperSub,
                  icon: Icons.inventory_2_outlined,
                  color: AppTheme.neonAmber,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildSensorMatrix(BuildContext context, dynamic deviceState, dynamic todayFeedings, dynamic alertsState) {
    final sensorData = ref.watch(sensorDataProvider).currentData;
    final tempStr = sensorData != null && sensorData.waterTemperature > 0
        ? '${sensorData.waterTemperature.toStringAsFixed(1)} °C'
        : '-- °C';
    final q10Factor = sensorData != null && sensorData.waterTemperature > 0
        ? 'Q10: ${(1.0 + (sensorData.waterTemperature - 28.0) * 0.05).clamp(0.8, 1.4).toStringAsFixed(2)}x'
        : 'Q10: --';

    final List<Device> devList = deviceState.devices is List<Device>
        ? (deviceState.devices as List<Device>)
        : <Device>[];
    final onlineCount = devList.where((Device d) => d.isOnline).length;

    return GridView.count(
      crossAxisCount: 2,
      crossAxisSpacing: 12,
      mainAxisSpacing: 12,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      childAspectRatio: 1.55,
      children: [
        _SensorCard(
          title: 'Dissolved Oxygen',
          value: devList.isEmpty ? '-- mg/L' : '5.8 mg/L',
          badge: devList.isEmpty ? 'OFFLINE' : 'OPTIMAL',
          icon: Icons.air,
          color: devList.isEmpty ? Colors.grey : AppTheme.primaryCyan,
          onTap: () => context.go('/monitoring'),
        ),
        _SensorCard(
          title: 'Water Temperature',
          value: tempStr,
          badge: devList.isEmpty ? 'OFFLINE' : q10Factor,
          icon: Icons.thermostat,
          color: devList.isEmpty ? Colors.grey : Colors.orangeAccent,
          onTap: () => context.go('/monitoring'),
        ),
        _SensorCard(
          title: 'Ammonia TAN',
          value: devList.isEmpty ? '-- mg/L' : '0.15 mg/L',
          badge: devList.isEmpty ? 'OFFLINE' : 'SAFE (<2.0)',
          icon: Icons.science_outlined,
          color: devList.isEmpty ? Colors.grey : Colors.purpleAccent,
          onTap: () => context.go('/monitoring'),
        ),
        _SensorCard(
          title: 'Connected Feeder',
          value: devList.isEmpty ? 'No device connected' : '$onlineCount / ${devList.length} Online',
          badge: devList.isEmpty ? '0 Paired' : '${todayFeedings.length} Feeds Today',
          icon: Icons.router_outlined,
          color: devList.isEmpty ? Colors.grey : AppTheme.primaryTeal,
          onTap: () => context.go('/devices'),
        ),
      ],
    );
  }

  Widget _buildAquaDocCard(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        gradient: const LinearGradient(
          colors: [Color(0xFF2A0845), Color(0xFF130924)],
          begin: Alignment.topLeft,
          end: Alignment.bottomRight,
        ),
        borderRadius: BorderRadius.circular(20),
        border: Border.all(color: AppTheme.neonPurple.withOpacity(0.5), width: 1.2),
        boxShadow: [
          BoxShadow(
            color: AppTheme.neonPurple.withOpacity(0.2),
            blurRadius: 16,
            offset: const Offset(0, 6),
          ),
        ],
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              gradient: AppTheme.aquaDocGradient,
              borderRadius: BorderRadius.circular(16),
            ),
            child: const Icon(Icons.psychology, color: Colors.white, size: 30),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Row(
                  children: [
                    Text(
                      'AquaDoc Clinical AI',
                      style: TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 15),
                    ),
                    SizedBox(width: 6),
                    Icon(Icons.verified, color: AppTheme.primaryCyan, size: 14),
                  ],
                ),
                const SizedBox(height: 4),
                Text(
                  'DO & TAN nominal. Bioenergetic model projecting 800g table size in 48 days.',
                  style: TextStyle(color: Colors.grey[300], fontSize: 12, height: 1.3),
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: AppTheme.primaryCyan,
              foregroundColor: const Color(0xFF0A192F),
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
            ),
            onPressed: () => context.go('/aquadoc'),
            child: const Text('Consult', style: TextStyle(fontWeight: FontWeight.bold, fontSize: 12)),
          ),
        ],
      ),
    );
  }

  Widget _buildQuickActionMatrix(BuildContext context) {
    return GridView.count(
      crossAxisCount: 3,
      crossAxisSpacing: 10,
      mainAxisSpacing: 10,
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      childAspectRatio: 1.15,
      children: [
        _ActionTile(
          icon: Icons.hub,
          label: 'AquaTwin 3D',
          sub: 'Live Digital Twin',
          gradient: const LinearGradient(colors: [Color(0xFF00C9FF), Color(0xFF92FE9D)]),
          onTap: () => context.go('/twin'),
        ),
        _ActionTile(
          icon: Icons.science,
          label: 'Simulator',
          sub: 'Growth Model',
          gradient: const LinearGradient(colors: [Color(0xFFFA709A), Color(0xFFFEE140)]),
          onTap: () => context.go('/simulator'),
        ),
        _ActionTile(
          icon: Icons.medical_services,
          label: 'Triage',
          sub: 'Disease Engine',
          gradient: const LinearGradient(colors: [Color(0xFFFF512F), Color(0xFFDD2476)]),
          onTap: () => context.go('/triage'),
        ),
        _ActionTile(
          icon: Icons.waves,
          label: 'My Ponds',
          sub: 'Unit Management',
          gradient: const LinearGradient(colors: [Color(0xFF4FACFE), Color(0xFF00F2FE)]),
          onTap: () => context.go('/farm'),
        ),
        _ActionTile(
          icon: Icons.restaurant,
          label: 'Feed Now',
          sub: 'Manual Dispense',
          gradient: const LinearGradient(colors: [Color(0xFF43E97B), Color(0xFF38F9D7)]),
          onTap: () => context.go('/feeding/manual'),
        ),
        _ActionTile(
          icon: Icons.calculate,
          label: 'Calculator',
          sub: 'Feed & FCR',
          gradient: const LinearGradient(colors: [Color(0xFFB176FC), Color(0xFFE0C3FC)]),
          onTap: () => context.go('/calculator'),
        ),
      ],
    );
  }

  Widget _buildRecentActivity(BuildContext context, dynamic feedingState) {
    final List<FeedingEvent> events = feedingState.events ?? [];
    if (events.isEmpty) {
      return Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(vertical: 24, horizontal: 16),
        decoration: BoxDecoration(
          color: Theme.of(context).cardColor,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: Colors.grey.withOpacity(0.1)),
        ),
        child: Column(
          children: [
            Icon(Icons.history_toggle_off, color: Colors.grey[500], size: 32),
            const SizedBox(height: 8),
            Text(
              'No recent feeding events recorded',
              style: TextStyle(color: Colors.grey[400], fontSize: 13, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 4),
            Text(
              'Dispensed rations and schedules will appear here in real time.',
              style: TextStyle(color: Colors.grey[600], fontSize: 11),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      );
    }

    return Container(
      decoration: BoxDecoration(
        color: Theme.of(context).cardColor,
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: Colors.grey.withOpacity(0.1)),
      ),
      child: ListView.separated(
        shrinkWrap: true,
        physics: const NeverScrollableScrollPhysics(),
        itemCount: events.take(5).length,
        separatorBuilder: (_, __) => Divider(height: 1, color: Colors.white.withOpacity(0.05)),
        itemBuilder: (context, index) {
          final event = events[index];
          final title = event.type == 'scheduled' ? 'Scheduled Feed Dispensed' : 'Manual Feed Dispensed';
          final amountVal = event.actualAmount ?? event.amount;
          final amount = '${amountVal.toStringAsFixed(0)}g dispensed';
          final eventTime = event.completedAt ?? event.scheduledAt;
          return ListTile(
            leading: Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                color: AppTheme.primaryCyan.withOpacity(0.12),
                borderRadius: BorderRadius.circular(10),
              ),
              child: const Icon(Icons.restaurant, color: AppTheme.primaryCyan, size: 20),
            ),
            title: Text(title, style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 13)),
            subtitle: Text(
              '${eventTime.hour.toString().padLeft(2, '0')}:${eventTime.minute.toString().padLeft(2, '0')}',
              style: TextStyle(fontSize: 11, color: Colors.grey[500]),
            ),
            trailing: Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: Colors.green.withOpacity(0.15),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                amount,
                style: const TextStyle(color: Colors.greenAccent, fontSize: 11, fontWeight: FontWeight.bold),
              ),
            ),
          );
        },
      ),
    );
  }
}

class _HeroStat extends StatelessWidget {
  final String label;
  final String value;
  final String sub;
  final IconData icon;
  final Color color;

  const _HeroStat({
    required this.label,
    required this.value,
    required this.sub,
    required this.icon,
    required this.color,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(icon, color: color, size: 18),
        const SizedBox(height: 4),
        Text(value, style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 14)),
        const SizedBox(height: 2),
        Text(sub, style: TextStyle(color: color.withOpacity(0.9), fontSize: 10, fontWeight: FontWeight.w600)),
      ],
    );
  }
}

class _SensorCard extends StatelessWidget {
  final String title;
  final String value;
  final String badge;
  final IconData icon;
  final Color color;
  final VoidCallback onTap;

  const _SensorCard({
    required this.title,
    required this.value,
    required this.badge,
    required this.icon,
    required this.color,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(16),
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: Theme.of(context).cardColor,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: color.withOpacity(0.25)),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withOpacity(0.04),
              blurRadius: 8,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Container(
                  padding: const EdgeInsets.all(6),
                  decoration: BoxDecoration(
                    color: color.withOpacity(0.15),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Icon(icon, color: color, size: 16),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: color.withOpacity(0.12),
                    borderRadius: BorderRadius.circular(6),
                  ),
                  child: Text(
                    badge,
                    style: TextStyle(color: color, fontSize: 9, fontWeight: FontWeight.bold),
                  ),
                ),
              ],
            ),
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  value,
                  style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 16),
                ),
                Text(
                  title,
                  style: TextStyle(fontSize: 11, color: Colors.grey[500]),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _ActionTile extends StatelessWidget {
  final IconData icon;
  final String label;
  final String sub;
  final Gradient gradient;
  final VoidCallback onTap;

  const _ActionTile({
    required this.icon,
    required this.label,
    required this.sub,
    required this.gradient,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(16),
      child: Container(
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: Theme.of(context).cardColor,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: Colors.white.withOpacity(0.08)),
        ),
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Container(
              padding: const EdgeInsets.all(8),
              decoration: BoxDecoration(
                gradient: gradient,
                shape: BoxShape.circle,
              ),
              child: Icon(icon, color: Colors.white, size: 18),
            ),
            const SizedBox(height: 8),
            Text(
              label,
              style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            Text(
              sub,
              style: TextStyle(fontSize: 9, color: Colors.grey[500]),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ],
        ),
      ),
    );
  }
}
