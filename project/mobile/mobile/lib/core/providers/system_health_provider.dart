import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../services/api_service.dart';
import 'auth_provider.dart';

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

/// Status of a single hardware component on the microcontroller.
class ComponentStatus {
  final String name;
  final String component;
  final String status; // "ok" | "error" | "neutral" | "skipped"
  final String message;

  const ComponentStatus({
    required this.name,
    required this.component,
    required this.status,
    required this.message,
  });

  factory ComponentStatus.fromJson(Map<String, dynamic> json) {
    return ComponentStatus(
      name: json['name'] ?? '',
      component: json['component'] ?? '',
      status: json['status'] ?? 'error',
      message: json['message'] ?? '',
    );
  }

  bool get isOk => status == 'ok';
  bool get isError => status == 'error';
  bool get isNeutral => status == 'neutral';
  bool get isSkipped => status == 'skipped';
}

/// Backend service health entry.
class BackendServiceHealth {
  final String name;
  final String status; // "ok" | "error"
  final String message;

  const BackendServiceHealth({
    required this.name,
    required this.status,
    required this.message,
  });

  factory BackendServiceHealth.fromJson(
    String name,
    Map<String, dynamic> json,
  ) {
    return BackendServiceHealth(
      name: name,
      status: json['status'] ?? 'error',
      message: json['message'] ?? '',
    );
  }

  bool get isOk => status == 'ok';
}

/// End-to-end pipeline connectivity.
class PipelineHealth {
  final bool? appToBackend;
  final bool? backendToApp;
  final bool? mcuToMqtt;
  final bool? mqttToBackend;
  final bool? backendToMqtt;
  final int? lastPingTime;

  const PipelineHealth({
    this.appToBackend,
    this.backendToApp,
    this.mcuToMqtt,
    this.mqttToBackend,
    this.backendToMqtt,
    this.lastPingTime,
  });

  factory PipelineHealth.fromJson(Map<String, dynamic> json) {
    return PipelineHealth(
      appToBackend: json['app_to_backend'] as bool?,
      backendToApp: json['backend_to_app'] as bool?,
      mcuToMqtt: json['mcu_to_mqtt'] as bool?,
      mqttToBackend: json['mqtt_to_backend'] as bool?,
      backendToMqtt: json['backend_to_mqtt'] as bool?,
      lastPingTime: json['last_ping_time'] as int?,
    );
  }

  /// Returns true if the full round-trip is verified.
  bool get isFullyConnected =>
      (appToBackend == true) &&
      (backendToApp == true) &&
      (mcuToMqtt == true) &&
      (mqttToBackend == true) &&
      (backendToMqtt == true);

  /// Returns true if at least app↔backend is working.
  bool get isPartiallyConnected =>
      (appToBackend == true) && (backendToApp == true);
}

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

class SystemHealthState {
  final String? deviceId;
  final List<ComponentStatus> components;
  final List<BackendServiceHealth> backendHealth;
  final PipelineHealth pipeline;
  final bool? canWorkWithoutCam;
  final int? diagnosticsTimestamp;
  final int? uptimeMs;
  final int? freeHeapBytes;
  final bool isLoading;
  final String? error;
  final String? message;
  final DateTime? lastFetched;

  const SystemHealthState({
    this.deviceId,
    this.components = const [],
    this.backendHealth = const [],
    this.pipeline = const PipelineHealth(),
    this.canWorkWithoutCam,
    this.diagnosticsTimestamp,
    this.uptimeMs,
    this.freeHeapBytes,
    this.isLoading = false,
    this.error,
    this.message,
    this.lastFetched,
  });

  SystemHealthState copyWith({
    String? deviceId,
    List<ComponentStatus>? components,
    List<BackendServiceHealth>? backendHealth,
    PipelineHealth? pipeline,
    bool? canWorkWithoutCam,
    int? diagnosticsTimestamp,
    int? uptimeMs,
    int? freeHeapBytes,
    bool? isLoading,
    String? error,
    String? message,
    DateTime? lastFetched,
  }) {
    return SystemHealthState(
      deviceId: deviceId ?? this.deviceId,
      components: components ?? this.components,
      backendHealth: backendHealth ?? this.backendHealth,
      pipeline: pipeline ?? this.pipeline,
      canWorkWithoutCam: canWorkWithoutCam ?? this.canWorkWithoutCam,
      diagnosticsTimestamp: diagnosticsTimestamp ?? this.diagnosticsTimestamp,
      uptimeMs: uptimeMs ?? this.uptimeMs,
      freeHeapBytes: freeHeapBytes ?? this.freeHeapBytes,
      isLoading: isLoading ?? this.isLoading,
      error: error,
      message: message,
      lastFetched: lastFetched ?? this.lastFetched,
    );
  }

  /// Count of components with OK status.
  int get okCount => components.where((c) => c.isOk).length;

  /// Count of components with error status.
  int get errorCount => components.where((c) => c.isError).length;

  /// True if no component is in error state.
  bool get allComponentsHealthy => errorCount == 0 && components.isNotEmpty;
}

// ---------------------------------------------------------------------------
// Notifier
// ---------------------------------------------------------------------------

class SystemHealthNotifier extends StateNotifier<SystemHealthState> {
  final ApiService _apiService;

  SystemHealthNotifier(this._apiService) : super(const SystemHealthState());

  Future<void> loadSystemHealth(String deviceId) async {
    if (!mounted) return;
    state = state.copyWith(isLoading: true, error: null, deviceId: deviceId);

    try {
      final response = await _apiService
          .getSystemHealth(deviceId)
          .timeout(const Duration(seconds: 20));
      if (!mounted) return;

      if (response.statusCode == 200) {
        final body = response.data;
        final data =
            body is Map<String, dynamic> ? (body['data'] ?? body) : body;
        if (data is! Map<String, dynamic>) {
          throw StateError('Invalid system health response');
        }

        // Parse components
        final List<ComponentStatus> components = [];
        final rawComponents = data['components'];
        if (rawComponents is List) {
          for (final c in rawComponents) {
            if (c is Map<String, dynamic>) {
              components.add(ComponentStatus.fromJson(c));
            }
          }
        }

        // Parse backend health
        final List<BackendServiceHealth> backendHealth = [];
        final rawBackend = data['backend_health'];
        if (rawBackend is Map<String, dynamic>) {
          for (final entry in rawBackend.entries) {
            if (entry.value is Map<String, dynamic>) {
              backendHealth.add(
                BackendServiceHealth.fromJson(entry.key, entry.value),
              );
            }
          }
        }

        // Parse pipeline
        PipelineHealth pipeline = const PipelineHealth();
        final rawPipeline = data['pipeline'];
        if (rawPipeline is Map<String, dynamic>) {
          pipeline = PipelineHealth.fromJson(rawPipeline);
        }

        state = state.copyWith(
          components: components,
          backendHealth: backendHealth,
          pipeline: pipeline,
          canWorkWithoutCam: data['can_work_without_cam'] as bool?,
          diagnosticsTimestamp: data['diagnostics_timestamp'] as int?,
          uptimeMs: data['uptime_ms'] as int?,
          freeHeapBytes: data['free_heap_bytes'] as int?,
          message: data['message'] as String?,
          isLoading: false,
          lastFetched: DateTime.now(),
        );
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Failed to load system health (${response.statusCode})',
        );
      }
    } catch (e) {
      if (!mounted) return;
      state = state.copyWith(
        isLoading: false,
        error: ApiService.describeError(
          e,
          fallback: 'Failed to load system health.',
        ),
      );
    }
  }

  Future<void> triggerDiagnostics(String deviceId) async {
    try {
      await _apiService.triggerDiagnostics(deviceId);
      // Reload after a short delay to get the fresh report
      await Future.delayed(const Duration(seconds: 3));
      if (!mounted) return;
      await loadSystemHealth(deviceId);
    } catch (e) {
      if (!mounted) return;
      state = state.copyWith(
        error: ApiService.describeError(
          e,
          fallback: 'Failed to trigger diagnostics.',
        ),
      );
    }
  }
}

// ---------------------------------------------------------------------------
// Providers
// ---------------------------------------------------------------------------

final systemHealthProvider =
    StateNotifierProvider.autoDispose<SystemHealthNotifier, SystemHealthState>((
      ref,
    ) {
      final apiService = ref.watch(apiServiceProvider);
      return SystemHealthNotifier(apiService);
    });

/// Convenience: are all hardware components healthy?
final allComponentsHealthyProvider = Provider.autoDispose<bool>((ref) {
  return ref.watch(systemHealthProvider).allComponentsHealthy;
});

/// Convenience: is the full E2E pipeline connected?
final pipelineFullyConnectedProvider = Provider.autoDispose<bool>((ref) {
  return ref.watch(systemHealthProvider).pipeline.isFullyConnected;
});
