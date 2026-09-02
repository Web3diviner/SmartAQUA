import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:equatable/equatable.dart';
import 'package:local_auth/local_auth.dart';

import '../services/api_service.dart';
import '../services/storage_service.dart';

const _authFieldUnset = Object();

// Auth State
class AuthState extends Equatable {
  final bool isAuthenticated;
  final bool isLoading;
  final String? userId;
  final String? userName;
  final String? email;
  final String? firstName;
  final String? lastName;
  final String? phoneNumber;
  final String? error;
  final String? statusMessage;
  final bool biometricAvailable;
  final bool biometricEnabled;

  const AuthState({
    this.isAuthenticated = false,
    this.isLoading = false,
    this.userId,
    this.userName,
    this.email,
    this.firstName,
    this.lastName,
    this.phoneNumber,
    this.error,
    this.statusMessage,
    this.biometricAvailable = false,
    this.biometricEnabled = false,
  });

  AuthState copyWith({
    bool? isAuthenticated,
    bool? isLoading,
    Object? userId = _authFieldUnset,
    Object? userName = _authFieldUnset,
    Object? email = _authFieldUnset,
    Object? firstName = _authFieldUnset,
    Object? lastName = _authFieldUnset,
    Object? phoneNumber = _authFieldUnset,
    Object? error = _authFieldUnset,
    Object? statusMessage = _authFieldUnset,
    bool? biometricAvailable,
    bool? biometricEnabled,
  }) {
    return AuthState(
      isAuthenticated: isAuthenticated ?? this.isAuthenticated,
      isLoading: isLoading ?? this.isLoading,
      userId:
          identical(userId, _authFieldUnset) ? this.userId : userId as String?,
      userName:
          identical(userName, _authFieldUnset)
              ? this.userName
              : userName as String?,
      email: identical(email, _authFieldUnset) ? this.email : email as String?,
      firstName:
          identical(firstName, _authFieldUnset)
              ? this.firstName
              : firstName as String?,
      lastName:
          identical(lastName, _authFieldUnset)
              ? this.lastName
              : lastName as String?,
      phoneNumber:
          identical(phoneNumber, _authFieldUnset)
              ? this.phoneNumber
              : phoneNumber as String?,
      error: identical(error, _authFieldUnset) ? this.error : error as String?,
      statusMessage:
          identical(statusMessage, _authFieldUnset)
              ? this.statusMessage
              : statusMessage as String?,
      biometricAvailable: biometricAvailable ?? this.biometricAvailable,
      biometricEnabled: biometricEnabled ?? this.biometricEnabled,
    );
  }

  @override
  List<Object?> get props => [
    isAuthenticated,
    isLoading,
    userId,
    userName,
    email,
    firstName,
    lastName,
    phoneNumber,
    error,
    statusMessage,
    biometricAvailable,
    biometricEnabled,
  ];
}

// Auth Notifier
class AuthNotifier extends StateNotifier<AuthState> {
  final ApiService _apiService;
  final LocalAuthentication _localAuth = LocalAuthentication();

  AuthNotifier(this._apiService) : super(const AuthState()) {
    _checkBiometricAvailability();
  }

  Future<void> _checkBiometricAvailability() async {
    try {
      final canCheckBiometrics = await _localAuth.canCheckBiometrics;
      final isDeviceSupported = await _localAuth.isDeviceSupported();
      final biometricEnabled = StorageService.getBiometricEnabled();

      state = state.copyWith(
        biometricAvailable: canCheckBiometrics && isDeviceSupported,
        biometricEnabled: biometricEnabled,
      );
    } catch (e) {
      state = state.copyWith(biometricAvailable: false);
    }
  }

  Future<bool> authenticateWithBiometrics() async {
    if (!state.biometricAvailable || !state.biometricEnabled) {
      return false;
    }

    try {
      final authenticated = await _localAuth.authenticate(
        localizedReason: 'Authenticate to access SmartAqua',
        options: const AuthenticationOptions(
          stickyAuth: true,
          biometricOnly: true,
        ),
      );

      if (authenticated) {
        // Check if we have stored credentials
        final token = await StorageService.getAccessToken();
        if (token != null) {
          final userId = StorageService.getUserId();
          state = state.copyWith(isAuthenticated: true, userId: userId);
          return true;
        }
      }
      return false;
    } catch (e) {
      return false;
    }
  }

  Future<void> setBiometricEnabled(bool enabled) async {
    await StorageService.setBiometricEnabled(enabled);
    state = state.copyWith(biometricEnabled: enabled);
  }

  Future<List<BiometricType>> getAvailableBiometrics() async {
    try {
      return await _localAuth.getAvailableBiometrics();
    } catch (e) {
      return [];
    }
  }

  Future<void> checkAuthStatus() async {
    state = state.copyWith(isLoading: true, error: null, statusMessage: null);

    try {
      final token = await StorageService.getAccessToken();
      if (token != null) {
        final userId = StorageService.getUserId();
        state = state.copyWith(
          isAuthenticated: true,
          isLoading: false,
          userId: userId,
        );
      } else {
        state = state.copyWith(
          isAuthenticated: false,
          isLoading: false,
          userId: null,
          userName: null,
          email: null,
          firstName: null,
          lastName: null,
          phoneNumber: null,
        );
      }
    } catch (e) {
      state = state.copyWith(
        isAuthenticated: false,
        isLoading: false,
        userId: null,
        userName: null,
        email: null,
        firstName: null,
        lastName: null,
        phoneNumber: null,
      );
    }
  }

  Future<bool> login(String email, String password) async {
    state = state.copyWith(isLoading: true, error: null, statusMessage: null);

    try {
      final response = await _apiService
          .login(email, password)
          .timeout(const Duration(seconds: 20));

      if (response.statusCode == 200) {
        final data = response.data;
        await _apiService.setAuthTokens(
          data['access_token'].toString(),
          data['refresh_token'].toString(),
        );

        state = state.copyWith(
          isAuthenticated: true,
          isLoading: false,
          email: email,
        );

        final profileLoaded = await loadProfile(showLoading: false);
        if (!profileLoaded) {
          final storedUserId = StorageService.getUserId();
          state = state.copyWith(
            isAuthenticated: true,
            isLoading: false,
            userId: storedUserId,
            userName: state.userName ?? email,
            email: state.email ?? email,
            error: null,
            statusMessage: null,
          );
        }
        return true;
      } else {
        state = state.copyWith(
          isLoading: false,
          error: 'Login failed. Please check your credentials.',
          statusMessage: null,
        );
        return false;
      }
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: _extractErrorMessage(
          e,
          fallback: 'Login failed. Please try again.',
        ),
        statusMessage: null,
      );
      return false;
    }
  }

  Future<bool> register(String name, String email, String password) async {
    state = state.copyWith(isLoading: true, error: null, statusMessage: null);

    try {
      final response = await _apiService
          .register(name, email, password)
          .timeout(const Duration(seconds: 20));

      if (response.statusCode == 201) {
        state = state.copyWith(
          isLoading: false,
          error: null,
          statusMessage: 'Registration successful. Please sign in.',
        );
        return true;
      } else {
        state = state.copyWith(
          isLoading: false,
          error: response.data['message'] ?? 'Registration failed.',
          statusMessage: null,
        );
        return false;
      }
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: _extractErrorMessage(
          e,
          fallback: 'Registration failed. Please try again.',
        ),
        statusMessage: null,
      );
      return false;
    }
  }

  Future<void> logout() async {
    try {
      await _apiService.logout();
    } catch (_) {}

    await _apiService.clearAuthTokens();
    state = state.copyWith(
      isAuthenticated: false,
      isLoading: false,
      userId: null,
      userName: null,
      email: null,
      firstName: null,
      lastName: null,
      phoneNumber: null,
      error: null,
      statusMessage: null,
    );
  }

  void clearError() {
    state = state.copyWith(error: null, statusMessage: null);
  }

  Future<bool> loadProfile({bool showLoading = true}) async {
    if (showLoading) {
      state = state.copyWith(isLoading: true, error: null, statusMessage: null);
    }

    try {
      final profileResponse = await _apiService.getProfile().timeout(
        const Duration(seconds: 10),
      );
      final data = profileResponse.data;
      final user = data is Map<String, dynamic> ? (data['user'] ?? data) : null;

      if (user is! Map<String, dynamic>) {
        state = state.copyWith(
          isLoading: false,
          error: 'Profile response was invalid.',
        );
        return false;
      }

      await _applyProfile(user);
      state = state.copyWith(
        isAuthenticated: true,
        isLoading: false,
        error: null,
        statusMessage: null,
      );
      return true;
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error:
            showLoading
                ? _extractErrorMessage(
                  e,
                  fallback: 'Failed to load your profile.',
                )
                : null,
      );
      return false;
    }
  }

  Future<bool> updateProfile({
    required String firstName,
    required String lastName,
    String? phoneNumber,
  }) async {
    state = state.copyWith(isLoading: true, error: null, statusMessage: null);

    try {
      final response = await _apiService.updateProfile(
        firstName: firstName,
        lastName: lastName,
        phoneNumber: phoneNumber,
      );

      final data = response.data;
      final user = data is Map<String, dynamic> ? (data['user'] ?? data) : null;

      if (user is Map<String, dynamic>) {
        await _applyProfile(user);
      }

      state = state.copyWith(
        isLoading: false,
        error: null,
        statusMessage: 'Profile updated successfully.',
      );
      return true;
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: _extractErrorMessage(
          e,
          fallback: 'Failed to update your profile.',
        ),
        statusMessage: null,
      );
      return false;
    }
  }

  // Password Reset
  Future<bool> requestPasswordReset(String email) async {
    state = state.copyWith(isLoading: true, error: null, statusMessage: null);

    try {
      final response = await _apiService.requestPasswordReset(email);
      final data = response.data;
      final resetCode =
          data is Map<String, dynamic> ? data['reset_code']?.toString() : null;
      final message =
          data is Map<String, dynamic> ? data['message']?.toString() : null;

      state = state.copyWith(
        isLoading: false,
        error: null,
        statusMessage:
            resetCode != null && resetCode.isNotEmpty
                ? 'Use reset code $resetCode to continue.'
                : (message ?? 'Reset request accepted.'),
      );
      return response.statusCode == 200;
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: _extractErrorMessage(
          e,
          fallback: 'Failed to request password reset.',
        ),
        statusMessage: null,
      );
      return false;
    }
  }

  Future<bool> verifyResetCode(String email, String code) async {
    state = state.copyWith(isLoading: true, error: null, statusMessage: null);

    try {
      final response = await _apiService.verifyResetCode(email, code);
      final data = response.data;
      final message =
          data is Map<String, dynamic> ? data['message']?.toString() : null;

      state = state.copyWith(
        isLoading: false,
        error: null,
        statusMessage: message ?? 'Reset code verified.',
      );
      return response.statusCode == 200;
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: _extractErrorMessage(
          e,
          fallback: 'Failed to verify reset code.',
        ),
        statusMessage: null,
      );
      return false;
    }
  }

  Future<bool> resetPassword(
    String email,
    String code,
    String newPassword,
  ) async {
    state = state.copyWith(isLoading: true, error: null, statusMessage: null);

    try {
      final response = await _apiService.resetPassword(
        email,
        code,
        newPassword,
      );
      final data = response.data;
      final message =
          data is Map<String, dynamic> ? data['message']?.toString() : null;

      state = state.copyWith(
        isLoading: false,
        error: null,
        statusMessage: message ?? 'Password reset successfully.',
      );
      return response.statusCode == 200;
    } catch (e) {
      state = state.copyWith(
        isLoading: false,
        error: _extractErrorMessage(e, fallback: 'Failed to reset password.'),
        statusMessage: null,
      );
      return false;
    }
  }

  String _extractErrorMessage(Object error, {required String fallback}) {
    return ApiService.describeError(error, fallback: fallback);
  }

  Future<void> _applyProfile(Map<String, dynamic> user) async {
    final userId = user['id']?.toString();
    final firstName = user['first_name']?.toString().trim();
    final lastName = user['last_name']?.toString().trim();
    final fullName =
        [
          if (firstName != null && firstName.isNotEmpty) firstName,
          if (lastName != null && lastName.isNotEmpty) lastName,
        ].join(' ').trim();
    final phoneNumber = user['phone_number']?.toString().trim();
    final email = user['email']?.toString();

    if (userId != null && userId.isNotEmpty) {
      await StorageService.setUserId(userId);
    }

    state = state.copyWith(
      userId: userId,
      firstName: firstName,
      lastName: lastName,
      phoneNumber:
          phoneNumber == null || phoneNumber.isEmpty ? null : phoneNumber,
      userName: fullName.isEmpty ? email : fullName,
      email: email,
    );
  }
}

// Providers
final apiServiceProvider = Provider<ApiService>((ref) => ApiService());

final authStateProvider = StateNotifierProvider<AuthNotifier, AuthState>((ref) {
  final apiService = ref.watch(apiServiceProvider);
  return AuthNotifier(apiService);
});
