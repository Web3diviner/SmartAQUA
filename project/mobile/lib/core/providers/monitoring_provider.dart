import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/sensor_data.dart';
import '../services/api_service.dart';
import 'auth_provider.dart';

// Sensor Data State
class SensorDataState {
  final SensorData? currentData;
  final bool isLoading;
  final String? error;
  final DateTime? lastUpdated;

  const SensorDataState({
    this.currentData,
    this.isLoading = false,
    this.error,
    this.lastUpdated,
  });

  SensorDataState copyWith({
    SensorData? currentData,
    bool? isLoading,
    String? error,
    DateTime? lastUpdated,
  }) {
    return SensorDataState(
      currentData: currentData ?? this.currentData,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      lastUpdated: lastUpdated ?? this.lastUpdated,
    );
  }
}

// Sensor Data Notifier
class SensorDataNotifier extends StateNotifier<SensorDataState> {
  final ApiService _apiService;

  SensorDataNotifier(this._apiService) : super(const SensorDataState());

  Future<void> loadSensorData(String deviceId) async {
    if (!mounted) return;
    state = state.copyWith(isLoading: true, error: null);

    try {
      final response = await _apiService
          .getSensorData(deviceId)
          .timeout(const Duration(seconds: 20));
      if (!mounted) return;
      if (response.statusCode == 200) {
        final body = response.data;
        final payload =
            body is Map<String, dynamic> ? (body['data'] ?? body) : body;
        final latest =
            payload is List && payload.isNotEmpty ? payload.first : payload;
        if (latest is! Map) {
          throw StateError('No sensor data available');
        }
        final data = SensorData.fromJson(Map<String, dynamic>.from(latest));
        state = state.copyWith(
          currentData: data,
          isLoading: false,
          lastUpdated: DateTime.now(),
        );
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Failed to load sensor data',
        );
      }
    } catch (e) {
      if (!mounted) return;
      state = state.copyWith(
        isLoading: false,
        error: ApiService.describeError(
          e,
          fallback: 'Failed to load sensor data.',
        ),
      );
    }
  }

  void updateFromMqtt(SensorData data) {
    if (!mounted) return;
    state = state.copyWith(currentData: data, lastUpdated: DateTime.now());
  }
}

// Sensor History State
class SensorHistoryState {
  final Map<String, SensorHistory> histories;
  final bool isLoading;
  final String? error;

  const SensorHistoryState({
    this.histories = const {},
    this.isLoading = false,
    this.error,
  });

  SensorHistoryState copyWith({
    Map<String, SensorHistory>? histories,
    bool? isLoading,
    String? error,
  }) {
    return SensorHistoryState(
      histories: histories ?? this.histories,
      isLoading: isLoading ?? this.isLoading,
      error: error,
    );
  }
}

// Sensor History Notifier
class SensorHistoryNotifier extends StateNotifier<SensorHistoryState> {
  final ApiService _apiService;

  SensorHistoryNotifier(this._apiService) : super(const SensorHistoryState());

  Future<void> loadHistory(
    String deviceId,
    String sensorType, {
    int hours = 24,
  }) async {
    if (!mounted) return;
    state = state.copyWith(isLoading: true, error: null);

    try {
      final response = await _apiService
          .getSensorHistory(deviceId, sensorType, hours: hours)
          .timeout(const Duration(seconds: 20));
      if (!mounted) return;
      if (response.statusCode == 200) {
        final body = response.data;
        final payload =
            body is Map<String, dynamic> ? (body['data'] ?? body) : body;
        final history = SensorHistory.fromJson(
          payload is Map<String, dynamic> ? payload : body,
        );
        state = state.copyWith(
          histories: {...state.histories, sensorType: history},
          isLoading: false,
        );
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Failed to load history',
        );
      }
    } catch (e) {
      if (!mounted) return;
      state = state.copyWith(
        isLoading: false,
        error: ApiService.describeError(
          e,
          fallback: 'Failed to load monitoring history.',
        ),
      );
    }
  }

  Future<void> loadAllHistories(String deviceId, {int hours = 24}) async {
    if (!mounted) return;
    state = state.copyWith(isLoading: true, error: null);

    try {
      final sensorTypes = ['temperature', 'feed_level', 'battery'];
      final Map<String, SensorHistory> histories = {};

      for (final type in sensorTypes) {
        try {
          final response = await _apiService
              .getSensorHistory(deviceId, type, hours: hours)
              .timeout(const Duration(seconds: 20));
          if (response.statusCode == 200) {
            final b = response.data;
            final p = b is Map<String, dynamic> ? (b['data'] ?? b) : b;
            histories[type] = SensorHistory.fromJson(
              p is Map<String, dynamic> ? p : b,
            );
          }
        } catch (_) {
          // Skip failed sensor types
        }
      }

      if (!mounted) return;
      state = state.copyWith(histories: histories, isLoading: false);
    } catch (e) {
      if (!mounted) return;
      state = state.copyWith(
        isLoading: false,
        error: ApiService.describeError(
          e,
          fallback: 'Failed to load monitoring history.',
        ),
      );
    }
  }
}

// Alerts State
class AlertsState {
  final List<DeviceAlert> alerts;
  final bool isLoading;
  final String? error;
  final int unreadCount;

  const AlertsState({
    this.alerts = const [],
    this.isLoading = false,
    this.error,
    this.unreadCount = 0,
  });

  AlertsState copyWith({
    List<DeviceAlert>? alerts,
    bool? isLoading,
    String? error,
    int? unreadCount,
  }) {
    return AlertsState(
      alerts: alerts ?? this.alerts,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      unreadCount: unreadCount ?? this.unreadCount,
    );
  }
}

// Alerts Notifier
class AlertsNotifier extends StateNotifier<AlertsState> {
  final ApiService _apiService;

  AlertsNotifier(this._apiService) : super(const AlertsState());

  Future<void> loadAlerts(String deviceId) async {
    if (!mounted) return;
    state = state.copyWith(isLoading: true, error: null);

    try {
      final response = await _apiService
          .getAlerts(deviceId)
          .timeout(const Duration(seconds: 20));
      if (!mounted) return;
      if (response.statusCode == 200) {
        final body = response.data;
        final payload =
            body is Map<String, dynamic>
                ? (body['data'] ?? body['alerts'] ?? body)
                : body;
        final List<dynamic> data = payload is List ? payload : [];
        final alerts = data.map((json) => DeviceAlert.fromJson(json)).toList();
        final unread = alerts.where((a) => !a.isRead).length;

        state = state.copyWith(
          alerts: alerts,
          isLoading: false,
          unreadCount: unread,
        );
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Failed to load alerts',
        );
      }
    } catch (e) {
      if (!mounted) return;
      state = state.copyWith(
        isLoading: false,
        error: ApiService.describeError(e, fallback: 'Failed to load alerts.'),
      );
    }
  }

  void addAlert(DeviceAlert alert) {
    if (!mounted) return;
    state = state.copyWith(
      alerts: [alert, ...state.alerts],
      unreadCount: state.unreadCount + 1,
    );
  }
}

// Providers
final sensorDataProvider =
    StateNotifierProvider.autoDispose<SensorDataNotifier, SensorDataState>((
      ref,
    ) {
      final apiService = ref.watch(apiServiceProvider);
      return SensorDataNotifier(apiService);
    });

final sensorHistoryProvider = StateNotifierProvider.autoDispose<
  SensorHistoryNotifier,
  SensorHistoryState
>((ref) {
  final apiService = ref.watch(apiServiceProvider);
  return SensorHistoryNotifier(apiService);
});

final alertsProvider =
    StateNotifierProvider.autoDispose<AlertsNotifier, AlertsState>((ref) {
      final apiService = ref.watch(apiServiceProvider);
      return AlertsNotifier(apiService);
    });

// Convenience providers
final currentTemperatureProvider = Provider.autoDispose<double?>((ref) {
  return ref.watch(sensorDataProvider).currentData?.waterTemperature;
});

final currentFeedLevelProvider = Provider.autoDispose<double?>((ref) {
  return ref.watch(sensorDataProvider).currentData?.feedLevel;
});

final currentBatteryProvider = Provider.autoDispose<double?>((ref) {
  return ref.watch(sensorDataProvider).currentData?.batteryLevel;
});

final unreadAlertsCountProvider = Provider.autoDispose<int>((ref) {
  return ref.watch(alertsProvider).unreadCount;
});

final criticalAlertsProvider = Provider.autoDispose<List<DeviceAlert>>((ref) {
  return ref
      .watch(alertsProvider)
      .alerts
      .where((a) => a.severity == AlertSeverity.critical)
      .toList();
});
