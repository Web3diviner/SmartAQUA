import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/sensor_data.dart';
import '../services/api_service.dart';
import 'auth_provider.dart';
import 'monitoring_provider.dart';

enum AppMqttState { disconnected, connecting, connected, error }

// Real-time connection state
class RealtimeState {
  final AppMqttState connectionState;
  final String? currentDeviceId;
  final DateTime? lastMessageAt;
  final String? error;

  const RealtimeState({
    this.connectionState = AppMqttState.disconnected,
    this.currentDeviceId,
    this.lastMessageAt,
    this.error,
  });

  RealtimeState copyWith({
    AppMqttState? connectionState,
    String? currentDeviceId,
    DateTime? lastMessageAt,
    String? error,
  }) {
    return RealtimeState(
      connectionState: connectionState ?? this.connectionState,
      currentDeviceId: currentDeviceId ?? this.currentDeviceId,
      lastMessageAt: lastMessageAt ?? this.lastMessageAt,
      error: error,
    );
  }

  bool get isConnected => connectionState == AppMqttState.connected;
}

// Real-time notifier
class RealtimeNotifier extends StateNotifier<RealtimeState> {
  final Ref _ref;
  final ApiService _apiService;
  Timer? _pollTimer;
  bool _pollInFlight = false;
  DateTime? _lastAlertsRefreshAt;

  static const Duration _pollInterval = Duration(seconds: 8);
  static const Duration _alertsRefreshInterval = Duration(seconds: 30);

  RealtimeNotifier(this._ref, this._apiService) : super(const RealtimeState());

  void _handleTelemetryPayload(Map<String, dynamic> payload) {
    try {
      final sensorData = SensorData.fromJson(payload);
      _ref.read(sensorDataProvider.notifier).updateFromMqtt(sensorData);
    } catch (e) {
      // Log error but don't crash
    }
  }

  Future<void> _pollCurrentDevice() async {
    final deviceId = state.currentDeviceId;
    if (deviceId == null || _pollInFlight) {
      return;
    }

    _pollInFlight = true;
    try {
      final response = await _apiService
          .getSensorData(deviceId)
          .timeout(const Duration(seconds: 20));

      if (response.statusCode == 200) {
        final body = response.data;
        final payload =
            body is Map<String, dynamic> ? (body['data'] ?? body) : body;
        final latest =
            payload is List && payload.isNotEmpty ? payload.first : payload;

        if (latest is Map) {
          _handleTelemetryPayload(Map<String, dynamic>.from(latest));
          state = state.copyWith(
            connectionState: AppMqttState.connected,
            lastMessageAt: DateTime.now(),
            error: null,
          );
        }
      }

      final now = DateTime.now();
      final shouldRefreshAlerts =
          _lastAlertsRefreshAt == null ||
          now.difference(_lastAlertsRefreshAt!) >= _alertsRefreshInterval;

      if (shouldRefreshAlerts) {
        await _ref.read(alertsProvider.notifier).loadAlerts(deviceId);
        _lastAlertsRefreshAt = now;
      }
    } catch (e) {
      state = state.copyWith(
        connectionState: AppMqttState.error,
        error: ApiService.describeError(
          e,
          fallback: 'Failed to fetch latest device data from backend.',
        ),
      );
    } finally {
      _pollInFlight = false;
    }
  }

  void _startPolling({bool runImmediately = false}) {
    _pollTimer?.cancel();
    _pollTimer = Timer.periodic(_pollInterval, (_) {
      unawaited(_pollCurrentDevice());
    });

    if (runImmediately) {
      unawaited(_pollCurrentDevice());
    }
  }

  Future<bool> connect() async {
    state = state.copyWith(
      connectionState: AppMqttState.connecting,
      error: null,
    );

    state = state.copyWith(
      connectionState: AppMqttState.connected,
      error: null,
    );
    _startPolling(runImmediately: true);
    return true;
  }

  void subscribeToDevice(String deviceId) {
    if (!state.isConnected) {
      connect();
    }

    state = state.copyWith(currentDeviceId: deviceId);
    _startPolling(runImmediately: true);
  }

  void unsubscribeFromDevice(String deviceId) {
    if (state.currentDeviceId == deviceId) {
      state = state.copyWith(currentDeviceId: null);
      _pollTimer?.cancel();
    }
  }

  void sendFeedCommand(String deviceId, double amount) {
    unawaited(_apiService.triggerManualFeed(deviceId, amount));
  }

  void sendEmergencyStop(String deviceId) {
    state = state.copyWith(
      error: 'Emergency stop requires backend control endpoint integration.',
    );
  }

  void requestVideoCapture(String deviceId) {
    unawaited(_apiService.requestVideoCapture(deviceId));
  }

  void disconnect() {
    _pollTimer?.cancel();
    _pollTimer = null;
    state = state.copyWith(
      connectionState: AppMqttState.disconnected,
      currentDeviceId: null,
    );
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    disconnect();
    super.dispose();
  }
}

// Providers
final realtimeProvider = StateNotifierProvider<RealtimeNotifier, RealtimeState>(
  (ref) {
    final apiService = ref.watch(apiServiceProvider);
    return RealtimeNotifier(ref, apiService);
  },
);

// Convenience providers
final isRealtimeConnectedProvider = Provider<bool>((ref) {
  return ref.watch(realtimeProvider).isConnected;
});

final realtimeConnectionStateProvider = Provider<AppMqttState>((ref) {
  return ref.watch(realtimeProvider).connectionState;
});
