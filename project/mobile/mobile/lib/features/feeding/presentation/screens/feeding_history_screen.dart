import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:intl/intl.dart';

import '../../../../core/services/api_service.dart';
import '../../../../core/theme/app_theme.dart';
import '../../../../core/models/device.dart';
import '../../../../core/models/feeding.dart';
import '../../../../core/providers/auth_provider.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/feeding_provider.dart';

class FeedingHistoryScreen extends ConsumerStatefulWidget {
  const FeedingHistoryScreen({super.key});

  @override
  ConsumerState<FeedingHistoryScreen> createState() =>
      _FeedingHistoryScreenState();
}

class _FeedingHistoryScreenState extends ConsumerState<FeedingHistoryScreen> {
  String? _selectedDeviceId;
  final ScrollController _scrollController = ScrollController();

  @override
  void initState() {
    super.initState();
    _loadData();
    _scrollController.addListener(_onScroll);
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _onScroll() {
    if (_scrollController.position.pixels >=
        _scrollController.position.maxScrollExtent - 200) {
      final state = ref.read(feedingHistoryProvider);
      if (!state.isLoading && state.hasMore && _selectedDeviceId != null) {
        ref
            .read(feedingHistoryProvider.notifier)
            .loadHistory(_selectedDeviceId!);
      }
    }
  }

  Future<void> _loadData() async {
    await ref.read(deviceListProvider.notifier).loadDevices();
    final devices = ref.read(devicesProvider);
    if (devices.isNotEmpty && _selectedDeviceId == null) {
      _selectedDeviceId = devices.first.id;
      await ref
          .read(feedingHistoryProvider.notifier)
          .loadHistory(_selectedDeviceId!, refresh: true);
    }
  }

  @override
  Widget build(BuildContext context) {
    final deviceState = ref.watch(deviceListProvider);
    final historyState = ref.watch(feedingHistoryProvider);
    final todayFeedings = ref.watch(todayFeedingsProvider);

    // Calculate summaries
    final todayTotal = todayFeedings.fold<double>(
      0,
      (sum, e) => sum + _releasedAmount(e),
    );
    final weekTotal = _calculateWeekTotal(historyState.events);
    final monthTotal = _calculateMonthTotal(historyState.events);

    return Scaffold(
      appBar: AppBar(
        title: const Text('Feeding History'),
        actions: [
          IconButton(
            icon: const Icon(Icons.download_outlined),
            tooltip: 'Export CSV',
            onPressed: _selectedDeviceId == null ? null : _exportHistory,
          ),
          IconButton(
            icon: const Icon(Icons.filter_list),
            onPressed: () => _showFilterDialog(context),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: () async {
          if (_selectedDeviceId != null) {
            await ref
                .read(feedingHistoryProvider.notifier)
                .loadHistory(_selectedDeviceId!, refresh: true);
          }
        },
        child: CustomScrollView(
          controller: _scrollController,
          slivers: [
            // Device selector and summary
            SliverToBoxAdapter(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  children: [
                    // Device selector
                    Card(
                      child: ListTile(
                        leading: const Icon(Icons.router),
                        title: Text(
                          _getSelectedDeviceName(deviceState.devices),
                        ),
                        trailing: const Icon(Icons.arrow_drop_down),
                        onTap:
                            () => _showDeviceSelector(
                              context,
                              deviceState.devices,
                            ),
                      ),
                    ),
                    const SizedBox(height: 16),

                    // Summary card
                    Card(
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: Row(
                          children: [
                            Expanded(
                              child: _SummaryItem(
                                label: 'Today',
                                value: _formatWeight(todayTotal),
                                icon: Icons.today,
                              ),
                            ),
                            Expanded(
                              child: _SummaryItem(
                                label: 'This Week',
                                value: _formatWeight(weekTotal),
                                icon: Icons.date_range,
                              ),
                            ),
                            Expanded(
                              child: _SummaryItem(
                                label: 'This Month',
                                value: _formatWeight(monthTotal),
                                icon: Icons.calendar_month,
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),

            // History list
            if (historyState.isLoading && historyState.events.isEmpty)
              const SliverFillRemaining(
                child: Center(child: CircularProgressIndicator()),
              )
            else if (historyState.events.isEmpty)
              SliverFillRemaining(
                child: Center(
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(Icons.history, size: 64, color: Colors.grey[400]),
                      const SizedBox(height: 16),
                      Text(
                        'No feeding history',
                        style: TextStyle(color: Colors.grey[600]),
                      ),
                    ],
                  ),
                ),
              )
            else
              SliverPadding(
                padding: const EdgeInsets.symmetric(horizontal: 16),
                sliver: SliverList(
                  delegate: SliverChildBuilderDelegate(
                    (context, index) {
                      if (index >= historyState.events.length) {
                        return historyState.hasMore
                            ? const Padding(
                              padding: EdgeInsets.all(16),
                              child: Center(child: CircularProgressIndicator()),
                            )
                            : null;
                      }

                      final event = historyState.events[index];
                      final showDateHeader =
                          index == 0 ||
                          !_isSameDay(
                            event.scheduledAt,
                            historyState.events[index - 1].scheduledAt,
                          );

                      return Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          if (showDateHeader) ...[
                            if (index > 0) const SizedBox(height: 16),
                            Text(
                              _formatDateHeader(event.scheduledAt),
                              style: Theme.of(context).textTheme.titleMedium
                                  ?.copyWith(fontWeight: FontWeight.bold),
                            ),
                            const SizedBox(height: 8),
                          ],
                          _HistoryItem(event: event),
                        ],
                      );
                    },
                    childCount:
                        historyState.events.length +
                        (historyState.hasMore ? 1 : 0),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }

  String _getSelectedDeviceName(List<Device> devices) {
    if (_selectedDeviceId == null) return 'Select a device';
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
          (ctx) => ListView(
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
                  trailing:
                      _selectedDeviceId == device.id
                          ? const Icon(Icons.check, color: Colors.green)
                          : null,
                  onTap: () {
                    setState(() => _selectedDeviceId = device.id);
                    ref
                        .read(feedingHistoryProvider.notifier)
                        .loadHistory(device.id, refresh: true);
                    Navigator.pop(ctx);
                  },
                ),
              ),
            ],
          ),
    );
  }

  void _showFilterDialog(BuildContext context) {
    showModalBottomSheet(
      context: context,
      builder:
          (ctx) => Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'Filter by Status',
                  style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                ),
                const SizedBox(height: 16),
                Wrap(
                  spacing: 8,
                  children: [
                    FilterChip(
                      label: const Text('All'),
                      selected: true,
                      onSelected: (_) {},
                    ),
                    FilterChip(
                      label: const Text('Completed'),
                      selected: false,
                      onSelected: (_) {},
                    ),
                    FilterChip(
                      label: const Text('Failed'),
                      selected: false,
                      onSelected: (_) {},
                    ),
                    FilterChip(
                      label: const Text('Manual'),
                      selected: false,
                      onSelected: (_) {},
                    ),
                    FilterChip(
                      label: const Text('Scheduled'),
                      selected: false,
                      onSelected: (_) {},
                    ),
                  ],
                ),
                const SizedBox(height: 24),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton(
                    onPressed: () => Navigator.pop(ctx),
                    child: const Text('Apply'),
                  ),
                ),
              ],
            ),
          ),
    );
  }

  Future<void> _exportHistory() async {
    final deviceId = _selectedDeviceId;
    if (deviceId == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Select a device before exporting.')),
      );
      return;
    }

    try {
      final response = await ref
          .read(apiServiceProvider)
          .exportFeedingHistory(deviceId)
          .timeout(const Duration(seconds: 20));
      final csv = response.data ?? '';
      if (csv.trim().isEmpty) {
        throw Exception('Export returned no data');
      }
      await Clipboard.setData(ClipboardData(text: csv));
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Feeding history CSV copied to clipboard.'),
        ),
      );
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            ApiService.describeError(
              e,
              fallback: 'Failed to export feeding history.',
            ),
          ),
        ),
      );
    }
  }

  double _calculateWeekTotal(List<FeedingEvent> events) {
    final weekAgo = DateTime.now().subtract(const Duration(days: 7));
    return events
        .where(
          (e) =>
              e.scheduledAt.isAfter(weekAgo) &&
              e.status == FeedingEventStatus.completed,
        )
        .fold<double>(0, (sum, e) => sum + _releasedAmount(e));
  }

  double _calculateMonthTotal(List<FeedingEvent> events) {
    final monthAgo = DateTime.now().subtract(const Duration(days: 30));
    return events
        .where(
          (e) =>
              e.scheduledAt.isAfter(monthAgo) &&
              e.status == FeedingEventStatus.completed,
        )
        .fold<double>(0, (sum, e) => sum + _releasedAmount(e));
  }

  double _releasedAmount(FeedingEvent event) {
    final actual = event.actualAmount;
    if (actual == null) return event.amount;
    if (actual == 0 &&
        event.status == FeedingEventStatus.completed &&
        event.amount > 0) {
      return event.amount;
    }
    return actual;
  }

  String _formatWeight(double grams) {
    if (grams >= 1000) {
      return '${(grams / 1000).toStringAsFixed(1)}kg';
    }
    return '${grams.toInt()}g';
  }

  bool _isSameDay(DateTime a, DateTime b) {
    return a.year == b.year && a.month == b.month && a.day == b.day;
  }

  String _formatDateHeader(DateTime date) {
    final now = DateTime.now();
    final today = DateTime(now.year, now.month, now.day);
    final yesterday = today.subtract(const Duration(days: 1));
    final dateOnly = DateTime(date.year, date.month, date.day);

    if (dateOnly == today) return 'Today';
    if (dateOnly == yesterday) return 'Yesterday';
    return DateFormat('EEEE, MMM d').format(date);
  }
}

class _SummaryItem extends StatelessWidget {
  final String label;
  final String value;
  final IconData icon;

  const _SummaryItem({
    required this.label,
    required this.value,
    required this.icon,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        Icon(icon, color: Theme.of(context).colorScheme.primary),
        const SizedBox(height: 8),
        Text(
          value,
          style: Theme.of(
            context,
          ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.bold),
        ),
        Text(
          label,
          style: Theme.of(
            context,
          ).textTheme.bodySmall?.copyWith(color: Colors.grey),
        ),
      ],
    );
  }
}

class _HistoryItem extends StatelessWidget {
  final FeedingEvent event;

  const _HistoryItem({required this.event});

  @override
  Widget build(BuildContext context) {
    final released = _releasedAmount();
    final requestedDiffers = (released - event.amount).abs() >= 0.05;
    final q10Factor = event.q10Factor;

    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: CircleAvatar(
          backgroundColor: _getStatusColor().withValues(alpha: 0.2),
          child: Icon(_getStatusIcon(), color: _getStatusColor()),
        ),
        title: Text('${_formatAmount(released)} released - ${_typeLabel()}'),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(DateFormat('h:mm a').format(event.scheduledAt)),
            if (requestedDiffers)
              Text(
                'Requested: ${_formatAmount(event.amount)}',
                style: TextStyle(color: Colors.grey[600], fontSize: 12),
              ),
            if (event.errorMessage != null)
              Text(
                event.errorMessage!,
                style: const TextStyle(
                  color: AppTheme.feedLevelLow,
                  fontSize: 12,
                ),
              ),
            if (event.waterTemperature != null)
              Text(
                'Temp: ${event.waterTemperature!.toStringAsFixed(1)} deg C',
                style: TextStyle(color: Colors.grey[600], fontSize: 12),
              ),
            if (q10Factor != null && (q10Factor - 1).abs() >= 0.01)
              Text(
                'Adaptive Q10: x${q10Factor.toStringAsFixed(2)}',
                style: TextStyle(color: Colors.grey[600], fontSize: 12),
              ),
          ],
        ),
        trailing: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              decoration: BoxDecoration(
                color: _getStatusColor().withValues(alpha: 0.2),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Text(
                _getStatusText(),
                style: TextStyle(color: _getStatusColor(), fontSize: 12),
              ),
            ),
            if (event.id.isNotEmpty) ...[
              const SizedBox(width: 4),
              IconButton(
                icon: const Icon(Icons.videocam_outlined, size: 20),
                tooltip: 'View Feeding Video',
                onPressed: () => context.go('/video/${event.id}'),
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(),
              ),
            ],
          ],
        ),
      ),
    );
  }

  double _releasedAmount() {
    final actual = event.actualAmount;
    if (actual == null) return event.amount;
    if (actual == 0 &&
        event.status == FeedingEventStatus.completed &&
        event.amount > 0) {
      return event.amount;
    }
    return actual;
  }

  String _formatAmount(double grams) {
    if (grams >= 1000) {
      return '${(grams / 1000).toStringAsFixed(1)}kg';
    }
    if (grams == grams.roundToDouble()) {
      return '${grams.toInt()}g';
    }
    return '${grams.toStringAsFixed(1)}g';
  }

  String _typeLabel() {
    final normalized = event.type.toLowerCase();
    if (normalized.contains('manual')) return 'Manual';
    if (normalized.contains('scheduled')) return 'Scheduled';
    if (normalized.contains('emergency')) return 'Emergency';
    return event.type;
  }

  Color _getStatusColor() {
    switch (event.status) {
      case FeedingEventStatus.completed:
        return AppTheme.feedLevelHigh;
      case FeedingEventStatus.failed:
        return AppTheme.feedLevelLow;
      case FeedingEventStatus.pending:
        return Colors.orange;
      case FeedingEventStatus.cancelled:
        return Colors.grey;
    }
  }

  IconData _getStatusIcon() {
    switch (event.status) {
      case FeedingEventStatus.completed:
        return Icons.check;
      case FeedingEventStatus.failed:
        return Icons.error_outline;
      case FeedingEventStatus.pending:
        return Icons.schedule;
      case FeedingEventStatus.cancelled:
        return Icons.cancel_outlined;
    }
  }

  String _getStatusText() {
    switch (event.status) {
      case FeedingEventStatus.completed:
        return 'Completed';
      case FeedingEventStatus.failed:
        return 'Failed';
      case FeedingEventStatus.pending:
        return 'Pending';
      case FeedingEventStatus.cancelled:
        return 'Cancelled';
    }
  }
}
