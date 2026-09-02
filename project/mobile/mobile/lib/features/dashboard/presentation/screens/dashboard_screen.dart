import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/models/feeding.dart';
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
    final issues = [
      if (deviceState.error != null) 'Devices: ${deviceState.error}',
      if (feedingState.error != null) 'Feeding: ${feedingState.error}',
      if (alertsState.error != null) 'Alerts: ${alertsState.error}',
    ];

    return Scaffold(
      appBar: AppBar(
        title: const Text('Dashboard'),
        actions: [
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
                      style: const TextStyle(color: Colors.white, fontSize: 10),
                    ),
                  ),
                ),
            ],
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _loadData,
        child:
            deviceState.isLoading
                ? Center(child: const CircularProgressIndicator())
                : SingleChildScrollView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      if (issues.isNotEmpty) ...[
                        _LoadIssueCard(
                          title: 'Some dashboard data failed to load',
                          issues: issues,
                          onRetry: _loadData,
                        ),
                        const SizedBox(height: 16),
                      ],
                      Text(
                        'Overview',
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      const SizedBox(height: 16),
                      Row(
                        children: [
                          Expanded(
                            child: _StatCard(
                              icon: Icons.devices,
                              label: 'Devices',
                              value: '${deviceState.devices.length}',
                              color: AppTheme.deviceOnline,
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: _StatCard(
                              icon: Icons.wifi,
                              label: 'Online',
                              value:
                                  '${ref.watch(onlineDevicesProvider).length}',
                              color: Theme.of(context).colorScheme.primary,
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 12),
                      Row(
                        children: [
                          Expanded(
                            child: _StatCard(
                              icon: Icons.restaurant,
                              label: 'Today\'s Feeds',
                              value: '${todayFeedings.length}',
                              color: AppTheme.feedLevelHigh,
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: _StatCard(
                              icon: Icons.warning_amber,
                              label: 'Alerts',
                              value: '${alertsState.unreadCount}',
                              color:
                                  alertsState.unreadCount > 0
                                      ? AppTheme.feedLevelLow
                                      : Colors.grey,
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 24),

                      Text(
                        'Quick Actions',
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      const SizedBox(height: 16),
                      Row(
                        children: [
                          Expanded(
                            child: _ActionButton(
                              icon: Icons.add_circle_outline,
                              label: 'Add Device',
                              onTap: () => context.go('/devices/pair'),
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: _ActionButton(
                              icon: Icons.restaurant,
                              label: 'Feed Now',
                              onTap: () => context.go('/feeding/manual'),
                            ),
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: _ActionButton(
                              icon: Icons.calculate,
                              label: 'Calculator',
                              onTap: () => context.go('/calculator'),
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 24),

                      Text(
                        'Recent Activity',
                        style: Theme.of(context).textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      const SizedBox(height: 16),

                      if (feedingState.isLoading)
                        const Center(child: CircularProgressIndicator())
                      else if (feedingState.events.isEmpty &&
                          alertsState.alerts.isEmpty)
                        Card(
                          child: Padding(
                            padding: const EdgeInsets.all(24),
                            child: Center(
                              child: Column(
                                children: [
                                  Icon(
                                    Icons.inbox,
                                    size: 48,
                                    color: Colors.grey[400],
                                  ),
                                  const SizedBox(height: 8),
                                  Text(
                                    'No recent activity',
                                    style: TextStyle(color: Colors.grey[600]),
                                  ),
                                ],
                              ),
                            ),
                          ),
                        )
                      else
                        ..._buildActivityItems(
                          feedingState.events,
                          alertsState.alerts,
                        ),
                    ],
                  ),
                ),
      ),
    );
  }

  List<Widget> _buildActivityItems(
    List<dynamic> feedings,
    List<dynamic> alerts,
  ) {
    final items = <_ActivityData>[];

    for (final event in feedings.take(5)) {
      items.add(
        _ActivityData(
          icon: Icons.restaurant,
          title: event.type == 'manual' ? 'Manual Feed' : 'Scheduled Feed',
          subtitle: '${event.amount}g dispensed',
          time: _formatTime(event.scheduledAt),
          timestamp: event.scheduledAt,
          isAlert: event.status == FeedingEventStatus.failed,
        ),
      );
    }

    for (final alert in alerts.take(3)) {
      items.add(
        _ActivityData(
          icon: _getAlertIcon(alert.alertType),
          title: alert.title,
          subtitle: alert.message,
          time: _formatTime(alert.createdAt),
          timestamp: alert.createdAt,
          isAlert: true,
        ),
      );
    }

    items.sort((a, b) => b.timestamp.compareTo(a.timestamp));

    return items
        .take(5)
        .map(
          (item) => _ActivityItem(
            icon: item.icon,
            title: item.title,
            subtitle: item.subtitle,
            time: item.time,
            isAlert: item.isAlert,
          ),
        )
        .toList();
  }

  IconData _getAlertIcon(String alertType) {
    switch (alertType) {
      case 'temperature':
        return Icons.thermostat;
      case 'battery':
        return Icons.battery_alert;
      case 'feed_level':
        return Icons.inventory_2;
      case 'oxygen':
        return Icons.air;
      default:
        return Icons.warning;
    }
  }

  String _formatTime(DateTime time) {
    final now = DateTime.now();
    final diff = now.difference(time);

    if (diff.inMinutes < 60) {
      return '${diff.inMinutes}m ago';
    } else if (diff.inHours < 24) {
      return '${diff.inHours}h ago';
    } else {
      return '${diff.inDays}d ago';
    }
  }
}

class _ActivityData {
  final IconData icon;
  final String title;
  final String subtitle;
  final String time;
  final DateTime timestamp;
  final bool isAlert;

  _ActivityData({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.time,
    required this.timestamp,
    this.isAlert = false,
  });
}

class _LoadIssueCard extends StatelessWidget {
  final String title;
  final List<String> issues;
  final Future<void> Function() onRetry;

  const _LoadIssueCard({
    required this.title,
    required this.issues,
    required this.onRetry,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      color: theme.colorScheme.errorContainer.withValues(alpha: 0.5),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              title,
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
                color: theme.colorScheme.onErrorContainer,
              ),
            ),
            const SizedBox(height: 8),
            ...issues.map(
              (issue) => Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Text(
                  issue,
                  style: TextStyle(color: theme.colorScheme.onErrorContainer),
                ),
              ),
            ),
            const SizedBox(height: 12),
            FilledButton.tonalIcon(
              onPressed: () {
                onRetry();
              },
              icon: const Icon(Icons.refresh),
              label: const Text('Retry'),
            ),
          ],
        ),
      ),
    );
  }
}

class _StatCard extends StatelessWidget {
  final IconData icon;
  final String label;
  final String value;
  final Color color;

  const _StatCard({
    required this.icon,
    required this.label,
    required this.value,
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
            Icon(icon, color: color, size: 28),
            const SizedBox(height: 12),
            Text(
              value,
              style: Theme.of(
                context,
              ).textTheme.headlineMedium?.copyWith(fontWeight: FontWeight.bold),
            ),
            Text(
              label,
              style: Theme.of(
                context,
              ).textTheme.bodyMedium?.copyWith(color: Colors.grey),
            ),
          ],
        ),
      ),
    );
  }
}

class _ActionButton extends StatelessWidget {
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  const _ActionButton({
    required this.icon,
    required this.label,
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
                size: 32,
                color: Theme.of(context).colorScheme.primary,
              ),
              const SizedBox(height: 8),
              Text(
                label,
                style: Theme.of(context).textTheme.bodySmall,
                textAlign: TextAlign.center,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ActivityItem extends StatelessWidget {
  final IconData icon;
  final String title;
  final String subtitle;
  final String time;
  final bool isAlert;

  const _ActivityItem({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.time,
    this.isAlert = false,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor:
              isAlert
                  ? AppTheme.feedLevelLow.withValues(alpha: 0.2)
                  : Theme.of(context).colorScheme.primaryContainer,
          child: Icon(
            icon,
            color:
                isAlert
                    ? AppTheme.feedLevelLow
                    : Theme.of(context).colorScheme.primary,
          ),
        ),
        title: Text(title),
        subtitle: Text(subtitle),
        trailing: Text(
          time,
          style: Theme.of(
            context,
          ).textTheme.bodySmall?.copyWith(color: Colors.grey),
        ),
      ),
    );
  }
}
