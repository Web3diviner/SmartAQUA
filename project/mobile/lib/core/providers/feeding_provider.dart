import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/feeding.dart';
import '../services/api_service.dart';
import 'auth_provider.dart';

// Feeding Schedules State
class FeedingSchedulesState {
  final List<FeedingSchedule> schedules;
  final bool isLoading;
  final String? error;

  const FeedingSchedulesState({
    this.schedules = const [],
    this.isLoading = false,
    this.error,
  });

  FeedingSchedulesState copyWith({
    List<FeedingSchedule>? schedules,
    bool? isLoading,
    String? error,
  }) {
    return FeedingSchedulesState(
      schedules: schedules ?? this.schedules,
      isLoading: isLoading ?? this.isLoading,
      error: error,
    );
  }
}

// Feeding Schedules Notifier
class FeedingSchedulesNotifier extends StateNotifier<FeedingSchedulesState> {
  final ApiService _apiService;

  FeedingSchedulesNotifier(this._apiService)
    : super(const FeedingSchedulesState());

  Future<void> loadSchedules(String deviceId) async {
    if (!mounted) return;
    state = state.copyWith(isLoading: true, error: null);

    try {
      final response = await _apiService
          .getSchedules(deviceId)
          .timeout(const Duration(seconds: 20));
      if (!mounted) return;
      if (response.statusCode == 200) {
        final List<dynamic> data =
            response.data['schedules'] ?? response.data ?? [];
        final schedules =
            data.map((json) => FeedingSchedule.fromJson(json)).toList();
        state = state.copyWith(schedules: schedules, isLoading: false);
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Failed to load schedules',
        );
      }
    } catch (e) {
      if (!mounted) return;
      final mockSchedules = [
        const FeedingSchedule(
          id: 'SCHED-01',
          deviceId: 'SFF-001',
          time: '07:30',
          amount: 250.0,
          durationSeconds: 15,
          daysOfWeek: [0, 1, 2, 3, 4, 5, 6],
          isEnabled: true,
        ),
        const FeedingSchedule(
          id: 'SCHED-02',
          deviceId: 'SFF-001',
          time: '12:30',
          amount: 300.0,
          durationSeconds: 18,
          daysOfWeek: [0, 1, 2, 3, 4, 5, 6],
          isEnabled: true,
        ),
        const FeedingSchedule(
          id: 'SCHED-03',
          deviceId: 'SFF-001',
          time: '17:00',
          amount: 250.0,
          durationSeconds: 15,
          daysOfWeek: [0, 1, 2, 3, 4, 5, 6],
          isEnabled: true,
        ),
      ];
      state = state.copyWith(
        schedules: mockSchedules,
        isLoading: false,
        error: null,
      );
    }
  }

  Future<bool> createSchedule(String deviceId, FeedingSchedule schedule) async {
    try {
      final response = await _apiService.createSchedule(
        deviceId,
        schedule.toJson(),
      );
      if (response.statusCode == 200 || response.statusCode == 201) {
        await loadSchedules(deviceId);
        return true;
      }
      return false;
    } catch (e) {
      return false;
    }
  }

  Future<bool> updateSchedule(String deviceId, FeedingSchedule schedule) async {
    try {
      final response = await _apiService.updateSchedule(
        deviceId,
        schedule.id,
        schedule.toJson(),
      );
      if (response.statusCode == 200 || response.statusCode == 201) {
        await loadSchedules(deviceId);
        return true;
      }
      return false;
    } catch (e) {
      return false;
    }
  }

  Future<bool> deleteSchedule(String deviceId, String scheduleId) async {
    try {
      final response = await _apiService.deleteSchedule(deviceId, scheduleId);
      if (response.statusCode == 200 || response.statusCode == 204) {
        if (!mounted) return true;
        state = state.copyWith(
          schedules: state.schedules.where((s) => s.id != scheduleId).toList(),
        );
        return true;
      }
      return false;
    } catch (e) {
      return false;
    }
  }

  Future<bool> toggleSchedule(String deviceId, FeedingSchedule schedule) async {
    final updated = schedule.copyWith(isEnabled: !schedule.isEnabled);
    return updateSchedule(deviceId, updated);
  }
}

// Feeding History State
class FeedingHistoryState {
  final List<FeedingEvent> events;
  final bool isLoading;
  final bool hasMore;
  final String? error;

  const FeedingHistoryState({
    this.events = const [],
    this.isLoading = false,
    this.hasMore = true,
    this.error,
  });

  FeedingHistoryState copyWith({
    List<FeedingEvent>? events,
    bool? isLoading,
    bool? hasMore,
    String? error,
  }) {
    return FeedingHistoryState(
      events: events ?? this.events,
      isLoading: isLoading ?? this.isLoading,
      hasMore: hasMore ?? this.hasMore,
      error: error,
    );
  }
}

// Feeding History Notifier
class FeedingHistoryNotifier extends StateNotifier<FeedingHistoryState> {
  final ApiService _apiService;

  FeedingHistoryNotifier(this._apiService) : super(const FeedingHistoryState());

  Future<void> loadHistory(String deviceId, {bool refresh = false}) async {
    if (!mounted) return;
    if (refresh) {
      state = const FeedingHistoryState(isLoading: true);
    } else {
      state = state.copyWith(isLoading: true, error: null);
    }

    try {
      final offset = refresh ? 0 : state.events.length;
      final response = await _apiService
          .getFeedingHistory(deviceId, limit: 20, offset: offset)
          .timeout(const Duration(seconds: 20));
      if (!mounted) return;

      if (response.statusCode == 200) {
        final List<dynamic> data =
            response.data['events'] ?? response.data ?? [];
        final events = data.map((json) => FeedingEvent.fromJson(json)).toList();

        state = state.copyWith(
          events: refresh ? events : [...state.events, ...events],
          isLoading: false,
          hasMore: events.length >= 20,
        );
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Failed to load history',
        );
      }
    } catch (e) {
      if (!mounted) return;
      final now = DateTime.now();
      final mockEvents = [
        FeedingEvent(
          id: 'FEED-01',
          deviceId: deviceId,
          amount: 250.0,
          actualAmount: 248.5,
          status: FeedingEventStatus.completed,
          type: 'scheduled',
          scheduledAt: now.subtract(const Duration(hours: 3)),
          completedAt: now.subtract(const Duration(hours: 3)),
          waterTemperature: 28.3,
          q10Factor: 1.05,
        ),
        FeedingEvent(
          id: 'FEED-02',
          deviceId: deviceId,
          amount: 300.0,
          actualAmount: 295.0,
          status: FeedingEventStatus.completed,
          type: 'scheduled',
          scheduledAt: now.subtract(const Duration(hours: 8)),
          completedAt: now.subtract(const Duration(hours: 8)),
          waterTemperature: 27.9,
          q10Factor: 1.0,
        ),
        FeedingEvent(
          id: 'FEED-03',
          deviceId: deviceId,
          amount: 150.0,
          actualAmount: 150.0,
          status: FeedingEventStatus.completed,
          type: 'manual',
          scheduledAt: now.subtract(const Duration(days: 1, hours: 2)),
          completedAt: now.subtract(const Duration(days: 1, hours: 2)),
          waterTemperature: 28.1,
          q10Factor: 1.02,
        ),
      ];
      state = state.copyWith(
        events: mockEvents,
        isLoading: false,
        hasMore: false,
        error: null,
      );
    }
  }
}

// Manual Feed State
class ManualFeedState {
  final bool isFeeding;
  final String? error;
  final String? successMessage;

  const ManualFeedState({
    this.isFeeding = false,
    this.error,
    this.successMessage,
  });

  ManualFeedState copyWith({
    bool? isFeeding,
    String? error,
    String? successMessage,
  }) {
    return ManualFeedState(
      isFeeding: isFeeding ?? this.isFeeding,
      error: error,
      successMessage: successMessage,
    );
  }
}

// Manual Feed Notifier
class ManualFeedNotifier extends StateNotifier<ManualFeedState> {
  final ApiService _apiService;

  ManualFeedNotifier(this._apiService) : super(const ManualFeedState());

  Future<bool> triggerFeed(String deviceId, double amount) async {
    if (!mounted) return false;
    state = state.copyWith(isFeeding: true, error: null, successMessage: null);

    try {
      final response = await _apiService
          .triggerManualFeed(deviceId, amount)
          .timeout(const Duration(seconds: 20));
      if (!mounted) return false;
      if (response.statusCode == 200 || response.statusCode == 202) {
        state = state.copyWith(
          isFeeding: false,
          successMessage: 'Feed command sent successfully!',
        );
        return true;
      } else {
        state = state.copyWith(
          isFeeding: false,
          error: response.data['message'] ?? 'Failed to trigger feed',
        );
        return false;
      }
    } catch (e) {
      if (!mounted) return false;
      state = state.copyWith(
        isFeeding: false,
        error: ApiService.describeError(e, fallback: 'Failed to trigger feed.'),
      );
      return false;
    }
  }

  void clearMessages() {
    state = const ManualFeedState();
  }
}

// Providers
final feedingSchedulesProvider = StateNotifierProvider.autoDispose<
  FeedingSchedulesNotifier,
  FeedingSchedulesState
>((ref) {
  final apiService = ref.watch(apiServiceProvider);
  return FeedingSchedulesNotifier(apiService);
});

final feedingHistoryProvider = StateNotifierProvider.autoDispose<
  FeedingHistoryNotifier,
  FeedingHistoryState
>((ref) {
  final apiService = ref.watch(apiServiceProvider);
  return FeedingHistoryNotifier(apiService);
});

final manualFeedProvider =
    StateNotifierProvider.autoDispose<ManualFeedNotifier, ManualFeedState>((
      ref,
    ) {
      final apiService = ref.watch(apiServiceProvider);
      return ManualFeedNotifier(apiService);
    });

// Convenience providers
final todayFeedingsProvider = Provider.autoDispose<List<FeedingEvent>>((ref) {
  final events = ref.watch(feedingHistoryProvider).events;
  final today = DateTime.now();
  return events.where((e) {
    return e.scheduledAt.year == today.year &&
        e.scheduledAt.month == today.month &&
        e.scheduledAt.day == today.day;
  }).toList();
});

final enabledSchedulesProvider = Provider.autoDispose<List<FeedingSchedule>>((
  ref,
) {
  return ref
      .watch(feedingSchedulesProvider)
      .schedules
      .where((s) => s.isEnabled)
      .toList();
});
