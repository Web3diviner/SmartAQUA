import 'dart:convert';
import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:crypto/crypto.dart';
import 'package:dio/dio.dart';
import 'package:dio/io.dart';
import 'package:logger/logger.dart';

import '../config/env_config.dart';
import 'storage_service.dart';

class ApiService {
  static final ValueNotifier<ApiDebugInfo> debugNotifier = ValueNotifier(
    ApiDebugInfo.initial(),
  );
  static const _redacted = '***';
  static const _sensitiveKeys = {
    'password',
    'new_password',
    'refresh_token',
    'access_token',
    'authorization',
    'token',
  };

  /// API base URL from environment configuration
  static String get baseUrl => EnvConfig.apiBaseUrl;

  /// Certificate fingerprints from environment configuration
  static List<String> get _pinnedCertificates => EnvConfig.pinnedCertificates;

  /// API domain for certificate validation
  static String get _apiDomain => EnvConfig.apiDomain;

  late final Dio _dio;
  final Logger _logger = Logger();
  String? _accessTokenCache;
  String? _refreshTokenCache;
  Future<String?>? _loadingAccessToken;
  Future<String?>? _loadingRefreshToken;
  static const Duration _tokenReadTimeout = Duration(seconds: 4);

  ApiService() {
    _dio = Dio(
      BaseOptions(
        baseUrl: baseUrl,
        connectTimeout: const Duration(seconds: 30),
        receiveTimeout: const Duration(seconds: 30),
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
      ),
    );

    // Configure certificate pinning
    _configureCertificatePinning();

    _dio.interceptors.add(
      InterceptorsWrapper(
        onRequest: _onRequest,
        onResponse: _onResponse,
        onError: _onError,
      ),
    );
  }

  /// Configure certificate pinning for enhanced security.
  ///
  /// Creates a single [HttpClient] that is reused across all requests so that
  /// TLS sessions are shared. Previously a *new* HttpClient was created on
  /// every request, which forced a fresh TLS handshake each time. On Render's
  /// free tier this caused connection stalls after a cold start because
  /// multiple simultaneous TLS negotiations overwhelmed the waking proxy.
  void _configureCertificatePinning() {
    // Create ONE HttpClient and reuse it for every request.
    final sharedClient = HttpClient();
    sharedClient.badCertificateCallback = (
      X509Certificate cert,
      String host,
      int port,
    ) {
      // Only validate certificates for our domain
      if (!host.contains(_apiDomain)) {
        return false;
      }

      // Skip pinning if no certificates configured
      if (_pinnedCertificates.isEmpty) {
        if (EnvConfig.isDevelopment) {
          _logger.w(
            'No certificate fingerprints configured - skipping pinning in dev mode',
          );
          return true;
        }
        _logger.e('No certificate fingerprints configured!');
        return false;
      }

      // Compute SHA256 fingerprint of the certificate
      final fingerprint = _computeCertificateFingerprint(cert);

      // Check if fingerprint matches any pinned certificate
      for (final pinned in _pinnedCertificates) {
        if (pinned.contains(fingerprint)) {
          return true;
        }
      }

      // In development mode, allow self-signed certificates
      if (EnvConfig.isDevelopment || EnvConfig.debugMode) {
        _logger.w('Certificate pinning bypassed in development mode');
        return true;
      }

      _logger.e('Certificate pinning failed for $host');
      return false;
    };

    (_dio.httpClientAdapter as IOHttpClientAdapter).createHttpClient = () {
      return sharedClient;
    };
  }

  /// Compute SHA256 fingerprint of X509 certificate
  String _computeCertificateFingerprint(X509Certificate cert) {
    // Get the DER-encoded certificate bytes
    final derBytes = cert.der;
    // Compute SHA256 hash
    final digest = sha256.convert(derBytes);
    // Return base64-encoded fingerprint
    return base64.encode(digest.bytes);
  }

  Future<void> _onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    debugNotifier.value = debugNotifier.value.copyWith(
      inFlight: true,
      method: options.method,
      path: options.path,
      startedAt: DateTime.now(),
      statusCode: null,
      errorType: null,
      errorMessage: null,
      finishedAt: null,
    );

    String? token;
    try {
      token = await _getAccessToken();
    } catch (e) {
      final message = 'Failed to read auth token from secure storage.';
      debugNotifier.value = debugNotifier.value.copyWith(
        inFlight: false,
        errorType: 'token_read',
        errorMessage: '$message ${e.toString()}',
        finishedAt: DateTime.now(),
      );
      handler.reject(
        DioException(
          requestOptions: options,
          type: DioExceptionType.unknown,
          error: e,
          message: message,
        ),
      );
      return;
    }

    if (token != null && token.isNotEmpty) {
      options.headers['Authorization'] = 'Bearer $token';
    } else {
      options.headers.remove('Authorization');
    }

    _logger.d(
      'REQUEST[${options.method}] => PATH: ${options.path} '
      'QUERY: ${_stringifyForLog(options.queryParameters)} '
      'DATA: ${_stringifyForLog(options.data)} '
      'AUTH: ${token != null && token.isNotEmpty} '
      'TOKEN_LEN: ${token?.length ?? 0}',
    );
    handler.next(options);
  }

  Future<String?> _getAccessToken() async {
    if (_accessTokenCache != null && _accessTokenCache!.isNotEmpty) {
      return _accessTokenCache;
    }

    _loadingAccessToken ??= StorageService.getAccessToken().timeout(
      _tokenReadTimeout,
    );

    try {
      final token = await _loadingAccessToken;
      if (token != null && token.isNotEmpty) {
        _accessTokenCache = token;
      }
      return token;
    } finally {
      _loadingAccessToken = null;
    }
  }

  Future<String?> _getRefreshToken() async {
    if (_refreshTokenCache != null && _refreshTokenCache!.isNotEmpty) {
      return _refreshTokenCache;
    }

    _loadingRefreshToken ??= StorageService.getRefreshToken().timeout(
      _tokenReadTimeout,
    );

    try {
      final token = await _loadingRefreshToken;
      if (token != null && token.isNotEmpty) {
        _refreshTokenCache = token;
      }
      return token;
    } finally {
      _loadingRefreshToken = null;
    }
  }

  void _onResponse(Response response, ResponseInterceptorHandler handler) {
    debugNotifier.value = debugNotifier.value.copyWith(
      inFlight: false,
      statusCode: response.statusCode,
      finishedAt: DateTime.now(),
    );
    _logger.d(
      'RESPONSE[${response.statusCode}] => PATH: ${response.requestOptions.path} '
      'DATA: ${_stringifyForLog(response.data)}',
    );
    handler.next(response);
  }

  Future<void> _onError(
    DioException err,
    ErrorInterceptorHandler handler,
  ) async {
    debugNotifier.value = debugNotifier.value.copyWith(
      inFlight: false,
      statusCode: err.response?.statusCode,
      errorType: err.type.name,
      errorMessage: err.message ?? 'Unknown error',
      finishedAt: DateTime.now(),
    );
    _logger.e(
      'ERROR[${err.response?.statusCode}] => PATH: ${err.requestOptions.path} '
      'TYPE: ${err.type.name} '
      'MESSAGE: ${err.message ?? 'Unknown error'} '
      'DATA: ${_stringifyForLog(err.response?.data)}',
    );

    if (err.response?.statusCode == 401) {
      // Try to refresh token
      final refreshed = await _refreshToken();
      if (refreshed) {
        // Retry the request
        final response = await _retry(err.requestOptions);
        handler.resolve(response);
        return;
      }
    }

    handler.next(err);
  }

  static String describeError(Object error, {String? fallback}) {
    if (error is DioException) {
      final data = error.response?.data;
      if (data is Map<String, dynamic>) {
        final message = data['error'] ?? data['message'] ?? data['details'];
        if (message is String && message.trim().isNotEmpty) {
          return message.trim();
        }
      }

      switch (error.type) {
        case DioExceptionType.connectionTimeout:
        case DioExceptionType.sendTimeout:
        case DioExceptionType.receiveTimeout:
          return 'Request timed out while contacting ${error.requestOptions.path}.';
        case DioExceptionType.connectionError:
          return 'Could not reach the backend. Check the API URL and network.';
        case DioExceptionType.badCertificate:
          return 'TLS certificate validation failed for the backend.';
        case DioExceptionType.cancel:
          return 'Request was cancelled.';
        case DioExceptionType.badResponse:
          final status = error.response?.statusCode;
          if (status == 401) {
            return 'Unauthorized (401). Sign in again and retry.';
          }
          if (status != null) {
            return 'Request failed with status $status on ${error.requestOptions.path}.';
          }
          break;
        case DioExceptionType.unknown:
          break;
      }

      if (error.message != null && error.message!.trim().isNotEmpty) {
        return error.message!.trim();
      }
    }

    final message = error.toString().trim();
    if (message.isNotEmpty && message != 'Exception') {
      return message;
    }

    return fallback ?? 'An unexpected error occurred.';
  }

  String _stringifyForLog(dynamic value) {
    final sanitized = _sanitizeForLog(value);
    final text = sanitized == null ? 'null' : sanitized.toString();
    if (text.length <= 400) {
      return text;
    }
    return '${text.substring(0, 400)}...';
  }

  dynamic _sanitizeForLog(dynamic value) {
    if (value is Map) {
      return value.map((key, dynamic nestedValue) {
        final normalizedKey = key.toString().toLowerCase();
        if (_sensitiveKeys.contains(normalizedKey)) {
          return MapEntry(key, _redacted);
        }
        return MapEntry(key, _sanitizeForLog(nestedValue));
      });
    }

    if (value is List) {
      return value.map(_sanitizeForLog).toList();
    }

    return value;
  }

  Future<bool> _refreshToken() async {
    try {
      final refreshToken = await _getRefreshToken();
      if (refreshToken == null || refreshToken.isEmpty) return false;

      final response = await Dio().post(
        '$baseUrl/auth/refresh',
        data: {'refresh_token': refreshToken},
      );

      if (response.statusCode == 200 || response.statusCode == 201) {
        final newAccessToken = response.data['access_token'];
        final newRefreshToken = response.data['refresh_token'];

        await setAuthTokens(
          newAccessToken.toString(),
          newRefreshToken.toString(),
        );
        _logger.i(
          'Token refresh applied. ACCESS_LEN=${newAccessToken.toString().length} '
          'REFRESH_LEN=${newRefreshToken.toString().length}',
        );

        return true;
      }
    } catch (e) {
      _logger.e('Token refresh failed: $e');
    }
    return false;
  }

  Future<Response> _retry(RequestOptions requestOptions) async {
    final token = _accessTokenCache ?? await StorageService.getAccessToken();
    final options = Options(
      method: requestOptions.method,
      headers: {...requestOptions.headers, 'Authorization': 'Bearer $token'},
    );
    return _dio.request(
      requestOptions.path,
      data: requestOptions.data,
      queryParameters: requestOptions.queryParameters,
      options: options,
    );
  }

  Map<String, dynamic> _withDeviceId(
    String deviceId, [
    Map<String, dynamic> additional = const {},
  ]) {
    return {'device_id': deviceId, ...additional};
  }

  Future<void> setAuthTokens(String accessToken, String refreshToken) async {
    _accessTokenCache = accessToken;
    _refreshTokenCache = refreshToken;
    await StorageService.setAccessToken(accessToken);
    await StorageService.setRefreshToken(refreshToken);
  }

  Future<void> clearAuthTokens() async {
    _accessTokenCache = null;
    _refreshTokenCache = null;
    _loadingAccessToken = null;
    _loadingRefreshToken = null;
    await StorageService.clearTokens();
    await StorageService.clearUserId();
  }

  Dio get dio => _dio;

  // Auth endpoints
  Future<Response> login(String email, String password) async {
    return _dio.post(
      '/auth/login',
      data: {'email': email, 'password': password},
    );
  }

  Future<Response> register(String name, String email, String password) async {
    final parts =
        name
            .trim()
            .split(RegExp(r'\s+'))
            .where((part) => part.isNotEmpty)
            .toList();
    final firstName = parts.isNotEmpty ? parts.first : name.trim();
    final lastName = parts.length > 1 ? parts.sublist(1).join(' ') : firstName;

    return _dio.post(
      '/auth/register',
      data: {
        'first_name': firstName,
        'last_name': lastName,
        'email': email,
        'password': password,
      },
    );
  }

  Future<Response> getProfile() async {
    return _dio.get('/users/profile');
  }

  Future<Response> updateProfile({
    required String firstName,
    required String lastName,
    String? phoneNumber,
  }) async {
    final normalizedPhone = phoneNumber?.trim();
    return _dio.put(
      '/users/profile',
      data: {
        'first_name': firstName,
        'last_name': lastName,
        'phone_number':
            normalizedPhone == null || normalizedPhone.isEmpty
                ? null
                : normalizedPhone,
      },
    );
  }

  Future<Response> logout() async {
    return _dio.post('/auth/logout');
  }

  // Device endpoints
  Future<Response> getDevices({CancelToken? cancelToken}) async {
    return _dio.get('/devices', cancelToken: cancelToken);
  }

  Future<Response> getDevice(String deviceId) async {
    return _dio.get('/devices/$deviceId');
  }

  Future<Response> bindDevice(
    String deviceSerial,
    String bindingCode,
    String name, {
    String? location,
  }) async {
    return _dio.post(
      '/devices/bind',
      data: {
        'device_serial': deviceSerial,
        'binding_code': bindingCode,
        'name': name,
        if (location != null && location.isNotEmpty) 'location': location,
      },
    );
  }

  Future<Response> unbindDevice(String deviceId) async {
    return _dio.delete('/devices/$deviceId');
  }

  // Feeding endpoints
  Future<Response> getSchedules(String deviceId) async {
    return _dio.get(
      '/feeding/schedules',
      queryParameters: _withDeviceId(deviceId),
    );
  }

  Future<Response> createSchedule(
    String deviceId,
    Map<String, dynamic> schedule,
  ) async {
    return _dio.post(
      '/feeding/schedules',
      data: {...schedule, 'device_id': deviceId},
    );
  }

  Future<Response> updateSchedule(
    String deviceId,
    String scheduleId,
    Map<String, dynamic> schedule,
  ) async {
    return _dio.put(
      '/feeding/schedules/$scheduleId',
      data: {...schedule, 'device_id': deviceId},
    );
  }

  Future<Response> deleteSchedule(String deviceId, String scheduleId) async {
    return _dio.delete('/feeding/schedules/$scheduleId');
  }

  Future<Response> triggerManualFeed(
    String deviceId,
    double amount, {
    int durationSeconds = 10,
  }) async {
    final safeDuration = durationSeconds > 0 ? durationSeconds : 10;

    return _dio.post(
      '/feeding/manual',
      data: {
        'device_id': deviceId,
        'quantity_grams': amount,
        'duration_seconds': safeDuration,
      },
    );
  }

  Future<Response> getFeedingHistory(
    String deviceId, {
    int? limit,
    int? offset,
  }) async {
    return _dio.get(
      '/feeding/history',
      queryParameters: {
        'device_id': deviceId,
        if (limit != null) 'limit': limit,
        if (offset != null) 'offset': offset,
      },
    );
  }

  Future<Response<String>> exportFeedingHistory(
    String deviceId, {
    int limit = 1000,
  }) async {
    return _dio.get<String>(
      '/feeding/history/export',
      queryParameters: {'device_id': deviceId, 'limit': limit},
      options: Options(responseType: ResponseType.plain),
    );
  }

  // Monitoring endpoints
  Future<Response> getSensorData(String deviceId) async {
    return _dio.get(
      '/monitoring/sensors',
      queryParameters: _withDeviceId(deviceId, {'limit': 1}),
    );
  }

  Future<Response> getSensorHistory(
    String deviceId,
    String sensorType, {
    int? hours,
  }) async {
    return _dio.get(
      '/monitoring/trends',
      queryParameters: _withDeviceId(deviceId, {
        'sensor_type': sensorType,
        if (hours != null) 'hours': hours,
      }),
    );
  }

  Future<Response> getAlerts(String deviceId) async {
    return _dio.get(
      '/monitoring/alerts',
      queryParameters: _withDeviceId(deviceId),
    );
  }

  // System Health endpoints
  Future<Response> getSystemHealth(String deviceId) async {
    return _dio.get('/devices/$deviceId/system-health');
  }

  Future<Response> triggerDiagnostics(String deviceId) async {
    return _dio.post('/devices/$deviceId/system-health/run');
  }

  // Calculator endpoints
  Future<Response> calculateFeed(Map<String, dynamic> params) async {
    return _dio.post('/calculator/recommend', data: params);
  }

  Future<Response> getSpecies({CancelToken? cancelToken}) async {
    return _dio.get('/calculator/species', cancelToken: cancelToken);
  }

  // Password reset endpoints
  Future<Response> requestPasswordReset(String email) async {
    return _dio.post('/auth/password-reset/request', data: {'email': email});
  }

  Future<Response> verifyResetCode(String email, String code) async {
    return _dio.post(
      '/auth/password-reset/verify',
      data: {'email': email, 'code': code},
    );
  }

  Future<Response> resetPassword(
    String email,
    String code,
    String newPassword,
  ) async {
    return _dio.post(
      '/auth/password-reset/confirm',
      data: {'email': email, 'code': code, 'new_password': newPassword},
    );
  }

  // Video verification endpoints
  Future<Response> getVideoClips(String deviceId, {int? limit}) async {
    return _dio.get(
      '/vision/clips/device/$deviceId',
      queryParameters: {if (limit != null) 'limit': limit},
    );
  }

  Future<Response> getVideoVerification(String feedingEventId) async {
    return _dio.get('/feeding-events/$feedingEventId/verification');
  }

  Future<Response> requestVideoCapture(String deviceId) async {
    return _dio.post('/devices/$deviceId/capture-video');
  }

  // FCR Analytics endpoints
  Future<Response> getFCRAnalytics(String deviceId, {int? days}) async {
    return _dio.get(
      '/fcr/$deviceId/analytics',
      queryParameters: {
        if (days != null)
          'start_date':
              DateTime.now()
                  .subtract(Duration(days: days))
                  .toIso8601String()
                  .split('T')
                  .first,
      },
    );
  }

  Future<Response> getGrowthPrediction(
    String deviceId, {
    required String species,
    required double currentWeight,
    required double targetWeight,
    int predictionDays = 30,
  }) async {
    return _dio.post(
      '/fcr/$deviceId/predict',
      data: {
        'species': species,
        'current_weight': currentWeight,
        'target_weight': targetWeight,
        'prediction_days': predictionDays,
      },
    );
  }
}

class ApiDebugInfo {
  final bool inFlight;
  final String? method;
  final String? path;
  final int? statusCode;
  final String? errorType;
  final String? errorMessage;
  final DateTime? startedAt;
  final DateTime? finishedAt;

  const ApiDebugInfo({
    required this.inFlight,
    this.method,
    this.path,
    this.statusCode,
    this.errorType,
    this.errorMessage,
    this.startedAt,
    this.finishedAt,
  });

  factory ApiDebugInfo.initial() => const ApiDebugInfo(inFlight: false);

  ApiDebugInfo copyWith({
    bool? inFlight,
    String? method,
    String? path,
    int? statusCode,
    String? errorType,
    String? errorMessage,
    DateTime? startedAt,
    DateTime? finishedAt,
  }) {
    return ApiDebugInfo(
      inFlight: inFlight ?? this.inFlight,
      method: method ?? this.method,
      path: path ?? this.path,
      statusCode: statusCode ?? this.statusCode,
      errorType: errorType ?? this.errorType,
      errorMessage: errorMessage ?? this.errorMessage,
      startedAt: startedAt ?? this.startedAt,
      finishedAt: finishedAt ?? this.finishedAt,
    );
  }
}
