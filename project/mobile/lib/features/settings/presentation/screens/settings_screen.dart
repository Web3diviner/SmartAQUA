import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../../core/config/env_config.dart';
import '../../../../core/providers/app_preferences_provider.dart';
import '../../../../core/providers/auth_provider.dart';
import '../../../../core/providers/realtime_provider.dart';
import '../../../../core/services/storage_service.dart';

class SettingsScreen extends ConsumerStatefulWidget {
  const SettingsScreen({super.key});

  @override
  ConsumerState<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends ConsumerState<SettingsScreen> {
  @override
  void initState() {
    super.initState();
    Future.microtask(
      () =>
          ref.read(authStateProvider.notifier).loadProfile(showLoading: false),
    );
  }

  Future<void> _applySettingChange(
    Future<void> Function() action,
    String successMessage,
  ) async {
    await action();
    if (!mounted) return;
    _showSnack(successMessage);
  }

  void _showSnack(String message) {
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }

  @override
  Widget build(BuildContext context) {
    final auth = ref.watch(authStateProvider);
    final prefs = ref.watch(appPreferencesProvider);
    final realtime = ref.watch(realtimeProvider);

    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        children: [
          if (auth.isLoading) const LinearProgressIndicator(),
          if (auth.error != null)
            _StatusBanner(
              message: auth.error!,
              isError: true,
              actionLabel: 'Reload profile',
              onAction: () {
                ref
                    .read(authStateProvider.notifier)
                    .loadProfile(showLoading: true);
              },
            ),
          if (auth.statusMessage != null)
            _StatusBanner(message: auth.statusMessage!, isError: false),
          _Header(auth: auth),
          _Section(
            title: 'Connection',
            children: [
              ListTile(
                leading: Icon(
                  realtime.isConnected ? Icons.cloud_done : Icons.cloud_off,
                  color: realtime.isConnected ? Colors.green : Colors.grey,
                ),
                title: const Text('Real-time Connection'),
                subtitle: Text(_status(realtime.connectionState)),
                trailing:
                    realtime.connectionState == AppMqttState.connecting
                        ? const SizedBox(
                          width: 20,
                          height: 20,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                        : Switch(
                          value: realtime.isConnected,
                          onChanged: (v) {
                            if (v) {
                              ref.read(realtimeProvider.notifier).connect();
                            } else {
                              ref.read(realtimeProvider.notifier).disconnect();
                            }
                          },
                        ),
              ),
              if (realtime.lastMessageAt != null)
                ListTile(
                  leading: const Icon(Icons.access_time),
                  title: const Text('Last Update'),
                  subtitle: Text(_lastUpdate(realtime.lastMessageAt!)),
                ),
            ],
          ),
          _Section(
            title: 'Account',
            children: [
              ListTile(
                leading: const Icon(Icons.person),
                title: const Text('Profile'),
                subtitle: Text(auth.email ?? 'View and edit your account'),
                trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                onTap: () => _showProfileDialog(context, auth),
              ),
              ListTile(
                leading: const Icon(Icons.security),
                title: const Text('Security'),
                subtitle: Text(
                  auth.biometricAvailable
                      ? (auth.biometricEnabled
                          ? 'Biometric sign-in enabled'
                          : 'Biometric sign-in disabled')
                      : 'Biometric sign-in unavailable',
                ),
                trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                onTap: () => _showSecuritySheet(context, auth),
              ),
            ],
          ),
          _Section(
            title: 'Notifications',
            children: [
              SwitchListTile(
                secondary: const Icon(Icons.notifications),
                title: const Text('Push Notifications'),
                subtitle: const Text('Receive alerts and updates'),
                value: prefs.notificationsEnabled,
                onChanged:
                    (v) => _applySettingChange(
                      () => ref
                          .read(appPreferencesProvider.notifier)
                          .setNotificationsEnabled(v),
                      'Push notifications updated.',
                    ),
              ),
              SwitchListTile(
                secondary: const Icon(Icons.warning),
                title: const Text('Alert Notifications'),
                subtitle: const Text('Low feed and health alerts'),
                value:
                    prefs.notificationsEnabled &&
                    prefs.alertNotificationsEnabled,
                onChanged:
                    prefs.notificationsEnabled
                        ? (v) => _applySettingChange(
                          () => ref
                              .read(appPreferencesProvider.notifier)
                              .setAlertNotificationsEnabled(v),
                          'Alert notifications updated.',
                        )
                        : null,
              ),
              SwitchListTile(
                secondary: const Icon(Icons.schedule),
                title: const Text('Feeding Reminders'),
                subtitle: const Text('Notify before scheduled feeds'),
                value:
                    prefs.notificationsEnabled && prefs.feedingRemindersEnabled,
                onChanged:
                    prefs.notificationsEnabled
                        ? (v) => _applySettingChange(
                          () => ref
                              .read(appPreferencesProvider.notifier)
                              .setFeedingRemindersEnabled(v),
                          'Feeding reminders updated.',
                        )
                        : null,
              ),
            ],
          ),
          _Section(
            title: 'Units & Display',
            children: [
              ListTile(
                leading: const Icon(Icons.thermostat),
                title: const Text('Temperature Unit'),
                subtitle: Text(_temperatureLabel(prefs.temperatureUnit)),
                trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                onTap:
                    () => _showPicker<String>(
                      context,
                      title: 'Temperature Unit',
                      current: prefs.temperatureUnit,
                      options: const {
                        'celsius': 'Celsius',
                        'fahrenheit': 'Fahrenheit',
                      },
                      onSelected:
                          (v) => ref
                              .read(appPreferencesProvider.notifier)
                              .setTemperatureUnit(v),
                    ),
              ),
              ListTile(
                leading: const Icon(Icons.scale),
                title: const Text('Weight Unit'),
                subtitle: Text(_weightLabel(prefs.weightUnit)),
                trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                onTap:
                    () => _showPicker<String>(
                      context,
                      title: 'Weight Unit',
                      current: prefs.weightUnit,
                      options: const {
                        'grams': 'Grams',
                        'ounces': 'Ounces',
                        'pounds': 'Pounds',
                      },
                      onSelected:
                          (v) => ref
                              .read(appPreferencesProvider.notifier)
                              .setWeightUnit(v),
                    ),
              ),
              SwitchListTile(
                secondary: Icon(
                  Theme.of(context).brightness == Brightness.dark
                      ? Icons.dark_mode_rounded
                      : Icons.light_mode_rounded,
                  color: Theme.of(context).brightness == Brightness.dark
                      ? const Color(0xFFFFD166)
                      : const Color(0xFF0077B6),
                ),
                title: const Text('Dark Mode'),
                subtitle: Text(
                  prefs.themeMode == ThemeMode.system
                      ? 'System (${Theme.of(context).brightness == Brightness.dark ? "Dark" : "Light"})'
                      : _themeLabel(prefs.themeMode),
                ),
                value: prefs.themeMode == ThemeMode.dark ||
                    (prefs.themeMode == ThemeMode.system &&
                        Theme.of(context).brightness == Brightness.dark),
                onChanged: (v) => ref
                    .read(appPreferencesProvider.notifier)
                    .setThemeMode(v ? ThemeMode.dark : ThemeMode.light),
              ),
              ListTile(
                leading: const Icon(Icons.palette),
                title: const Text('Theme Mode Preference'),
                subtitle: Text(_themeLabel(prefs.themeMode)),
                trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                onTap:
                    () => _showPicker<ThemeMode>(
                      context,
                      title: 'Theme Mode Preference',
                      current: prefs.themeMode,
                      options: const {
                        ThemeMode.system: 'System default',
                        ThemeMode.light: 'Light',
                        ThemeMode.dark: 'Dark',
                      },
                      onSelected:
                          (v) => ref
                              .read(appPreferencesProvider.notifier)
                              .setThemeMode(v),
                    ),
              ),
            ],
          ),
          _Section(
            title: 'Data & Storage',
            children: [
              ListTile(
                leading: const Icon(Icons.storage),
                title: const Text('Clear App Cache'),
                subtitle: const Text('Clear image cache and remembered device'),
                onTap: () => _showClearCacheDialog(context),
              ),
              ListTile(
                leading: const Icon(Icons.download),
                title: const Text('Export Settings Summary'),
                subtitle: const Text('Copy a diagnostic summary to clipboard'),
                onTap: () => _exportSummary(context, auth, prefs, realtime),
              ),
            ],
          ),
          _Section(
            title: 'Diagnostics',
            children: [
              ListTile(
                leading: const Icon(Icons.cloud_outlined),
                title: const Text('Backend URL'),
                subtitle: Text(EnvConfig.apiBaseUrl),
              ),
              ListTile(
                leading: const Icon(Icons.badge_outlined),
                title: const Text('Authenticated User'),
                subtitle: Text(auth.email ?? 'No authenticated user'),
              ),
              ListTile(
                leading: const Icon(Icons.perm_identity_outlined),
                title: const Text('User ID'),
                subtitle: Text(auth.userId ?? 'Not loaded'),
              ),
              ListTile(
                leading: const Icon(Icons.bug_report_outlined),
                title: const Text('Reload Profile'),
                subtitle: const Text('Force a fresh /users/profile request'),
                trailing: const Icon(Icons.refresh),
                onTap: () async {
                  final ok = await ref
                      .read(authStateProvider.notifier)
                      .loadProfile(showLoading: true);
                  if (!mounted) return;
                  _showSnack(
                    ok ? 'Profile reloaded.' : 'Profile reload failed.',
                  );
                },
              ),
            ],
          ),
          _Section(
            title: 'About',
            children: [
              const ListTile(
                leading: Icon(Icons.info),
                title: Text('App Version'),
                subtitle: Text('1.0.0 (Build 1)'),
              ),
              ListTile(
                leading: const Icon(Icons.description),
                title: const Text('Terms of Service'),
                trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                onTap:
                    () => _showText(
                      context,
                      'Terms of Service',
                      'Validate schedules, calibration and device safety before deployment.',
                    ),
              ),
              ListTile(
                leading: const Icon(Icons.privacy_tip),
                title: const Text('Privacy Policy'),
                trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                onTap:
                    () => _showText(
                      context,
                      'Privacy Policy',
                      'This app stores tokens securely on-device and saves local preferences such as theme and units.',
                    ),
              ),
              ListTile(
                leading: const Icon(Icons.help),
                title: const Text('Help & Support'),
                trailing: const Icon(Icons.arrow_forward_ios, size: 16),
                onTap: () => _showSupport(context, auth, prefs, realtime),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: OutlinedButton.icon(
              onPressed: () async {
                final confirm = await _confirmLogout(context);
                if (confirm == true) {
                  ref.read(realtimeProvider.notifier).disconnect();
                  await ref.read(authStateProvider.notifier).logout();
                  if (context.mounted) context.go('/login');
                }
              },
              icon: const Icon(Icons.logout, color: Colors.red),
              label: const Text(
                'Sign Out',
                style: TextStyle(color: Colors.red),
              ),
            ),
          ),
          const SizedBox(height: 32),
        ],
      ),
    );
  }

  String _status(AppMqttState state) {
    switch (state) {
      case AppMqttState.connected:
        return 'Connected to server';
      case AppMqttState.connecting:
        return 'Connecting...';
      case AppMqttState.disconnected:
        return 'Disconnected';
      case AppMqttState.error:
        return 'Connection error';
    }
  }

  String _lastUpdate(DateTime time) {
    final diff = DateTime.now().difference(time);
    if (diff.inSeconds < 60) return '${diff.inSeconds} seconds ago';
    if (diff.inMinutes < 60) return '${diff.inMinutes} minutes ago';
    return '${diff.inHours} hours ago';
  }

  String _themeLabel(ThemeMode mode) =>
      mode == ThemeMode.light
          ? 'Light'
          : mode == ThemeMode.dark
          ? 'Dark'
          : 'System default';

  String _temperatureLabel(String unit) =>
      unit == 'fahrenheit' ? 'Fahrenheit' : 'Celsius';

  String _weightLabel(String unit) {
    switch (unit) {
      case 'ounces':
        return 'Ounces';
      case 'pounds':
        return 'Pounds';
      default:
        return 'Grams';
    }
  }

  Future<void> _showProfileDialog(BuildContext context, AuthState auth) async {
    final parts = _splitName(auth);
    final first = TextEditingController(text: auth.firstName ?? parts.$1);
    final last = TextEditingController(text: auth.lastName ?? parts.$2);
    final phone = TextEditingController(text: auth.phoneNumber ?? '');
    var saving = false;
    await showDialog<void>(
      context: context,
      builder:
          (ctx) => StatefulBuilder(
            builder:
                (ctx, setState) => AlertDialog(
                  title: const Text('Profile'),
                  content: SingleChildScrollView(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        TextField(
                          controller: first,
                          decoration: const InputDecoration(
                            labelText: 'First Name',
                          ),
                        ),
                        const SizedBox(height: 12),
                        TextField(
                          controller: last,
                          decoration: const InputDecoration(
                            labelText: 'Last Name',
                          ),
                        ),
                        const SizedBox(height: 12),
                        TextField(
                          controller: phone,
                          keyboardType: TextInputType.phone,
                          decoration: const InputDecoration(
                            labelText: 'Phone Number',
                          ),
                        ),
                        const SizedBox(height: 12),
                        Align(
                          alignment: Alignment.centerLeft,
                          child: Text(auth.email ?? 'No email loaded'),
                        ),
                      ],
                    ),
                  ),
                  actions: [
                    TextButton(
                      onPressed: saving ? null : () => Navigator.pop(ctx),
                      child: const Text('Cancel'),
                    ),
                    FilledButton(
                      onPressed:
                          saving
                              ? null
                              : () async {
                                if (first.text.trim().isEmpty ||
                                    last.text.trim().isEmpty) {
                                  if (mounted) {
                                    _showSnack(
                                      'First and last name are required.',
                                    );
                                  }
                                  return;
                                }
                                setState(() => saving = true);
                                final ok = await ref
                                    .read(authStateProvider.notifier)
                                    .updateProfile(
                                      firstName: first.text.trim(),
                                      lastName: last.text.trim(),
                                      phoneNumber: phone.text.trim(),
                                    );
                                if (!ctx.mounted) return;
                                setState(() => saving = false);
                                if (ok) {
                                  Navigator.pop(ctx);
                                  if (mounted) {
                                    _showSnack('Profile updated.');
                                  }
                                }
                              },
                      child:
                          saving
                              ? const SizedBox(
                                width: 18,
                                height: 18,
                                child: CircularProgressIndicator(
                                  strokeWidth: 2,
                                ),
                              )
                              : const Text('Save'),
                    ),
                  ],
                ),
          ),
    );
    first.dispose();
    last.dispose();
    phone.dispose();
  }

  void _showSecuritySheet(BuildContext context, AuthState auth) {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder:
          (ctx) => Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Security', style: Theme.of(ctx).textTheme.titleLarge),
                SwitchListTile.adaptive(
                  contentPadding: EdgeInsets.zero,
                  secondary: Icon(
                    auth.biometricAvailable
                        ? Icons.fingerprint
                        : Icons.no_encryption_gmailerrorred,
                  ),
                  title: const Text('Biometric Sign-In'),
                  subtitle: Text(
                    auth.biometricAvailable
                        ? 'Use fingerprint or face unlock on this device.'
                        : 'This device does not report biometric support.',
                  ),
                  value: auth.biometricAvailable && auth.biometricEnabled,
                  onChanged:
                      auth.biometricAvailable
                          ? (v) async {
                            await ref
                                .read(authStateProvider.notifier)
                                .setBiometricEnabled(v);
                            if (!mounted) return;
                            _showSnack(
                              v
                                  ? 'Biometric sign-in enabled.'
                                  : 'Biometric sign-in disabled.',
                            );
                          }
                          : null,
                ),
                ListTile(
                  contentPadding: EdgeInsets.zero,
                  leading: const Icon(Icons.verified_user_outlined),
                  title: const Text('Session'),
                  subtitle: Text(
                    auth.isAuthenticated
                        ? 'Signed in as ${auth.email ?? 'current user'}'
                        : 'Not signed in',
                  ),
                ),
              ],
            ),
          ),
    );
  }

  Future<void> _showPicker<T>(
    BuildContext context, {
    required String title,
    required T current,
    required Map<T, String> options,
    required Future<void> Function(T value) onSelected,
  }) async {
    final selected = await showModalBottomSheet<T>(
      context: context,
      showDragHandle: true,
      builder:
          (ctx) => Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.all(16),
                child: Text(
                  title,
                  style: const TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
              ...options.entries.map(
                (entry) => ListTile(
                  title: Text(entry.value),
                  trailing:
                      entry.key == current
                          ? const Icon(Icons.check, color: Colors.green)
                          : null,
                  onTap: () => Navigator.pop(ctx, entry.key),
                ),
              ),
              const SizedBox(height: 16),
            ],
          ),
    );

    if (selected == null) return;
    await onSelected(selected);
    if (!mounted) return;
    _showSnack('$title updated.');
  }

  void _showClearCacheDialog(BuildContext context) {
    showDialog<void>(
      context: context,
      builder:
          (ctx) => AlertDialog(
            title: const Text('Clear App Cache'),
            content: const Text(
              'This clears the image cache and remembered device selection.',
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text('Cancel'),
              ),
              FilledButton(
                onPressed: () async {
                  await StorageService.clearDeviceId();
                  PaintingBinding.instance.imageCache.clear();
                  PaintingBinding.instance.imageCache.clearLiveImages();
                  if (!ctx.mounted) return;
                  Navigator.pop(ctx);
                  if (mounted) {
                    _showSnack('App cache cleared.');
                  }
                },
                child: const Text('Clear'),
              ),
            ],
          ),
    );
  }

  Future<void> _exportSummary(
    BuildContext context,
    AuthState auth,
    AppPreferencesState prefs,
    RealtimeState realtime,
  ) async {
    final text =
        StringBuffer()
          ..writeln('SmartAqua Settings Summary')
          ..writeln('Account: ${auth.email ?? 'Unknown'}')
          ..writeln('Name: ${auth.userName ?? 'Unknown'}')
          ..writeln('Theme: ${_themeLabel(prefs.themeMode)}')
          ..writeln(
            'Temperature Unit: ${_temperatureLabel(prefs.temperatureUnit)}',
          )
          ..writeln('Weight Unit: ${_weightLabel(prefs.weightUnit)}')
          ..writeln('Push Notifications: ${prefs.notificationsEnabled}')
          ..writeln('Alert Notifications: ${prefs.alertNotificationsEnabled}')
          ..writeln('Feeding Reminders: ${prefs.feedingRemindersEnabled}')
          ..writeln('Realtime Connection: ${_status(realtime.connectionState)}')
          ..writeln('API Base URL: ${EnvConfig.apiBaseUrl}')
          ..writeln('Auth Error: ${auth.error ?? 'None'}')
          ..writeln('Status Message: ${auth.statusMessage ?? 'None'}')
          ..writeln('Mobile Debug Mode: ${EnvConfig.debugMode}');
    await Clipboard.setData(ClipboardData(text: text.toString()));
    if (!context.mounted) return;
    _showSnack('Settings summary copied to clipboard.');
  }

  void _showSupport(
    BuildContext context,
    AuthState auth,
    AppPreferencesState prefs,
    RealtimeState realtime,
  ) {
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      builder:
          (ctx) => Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  'Help & Support',
                  style: Theme.of(ctx).textTheme.titleLarge,
                ),
                const SizedBox(height: 12),
                Text('Backend: ${EnvConfig.apiBaseUrl}'),
                Text('Signed in: ${auth.email ?? 'Unknown'}'),
                Text('Realtime: ${_status(realtime.connectionState)}'),
                const SizedBox(height: 12),
                FilledButton.icon(
                  onPressed:
                      () => _exportSummary(context, auth, prefs, realtime),
                  icon: const Icon(Icons.copy),
                  label: const Text('Copy Diagnostic Summary'),
                ),
              ],
            ),
          ),
    );
  }

  void _showText(BuildContext context, String title, String content) {
    showDialog<void>(
      context: context,
      builder:
          (ctx) => AlertDialog(
            title: Text(title),
            content: Text(content),
            actions: [
              FilledButton(
                onPressed: () => Navigator.pop(ctx),
                child: const Text('Close'),
              ),
            ],
          ),
    );
  }

  Future<bool?> _confirmLogout(BuildContext context) {
    return showDialog<bool>(
      context: context,
      builder:
          (ctx) => AlertDialog(
            title: const Text('Sign Out'),
            content: const Text('Are you sure you want to sign out?'),
            actions: [
              TextButton(
                onPressed: () => Navigator.pop(ctx, false),
                child: const Text('Cancel'),
              ),
              TextButton(
                onPressed: () => Navigator.pop(ctx, true),
                child: const Text(
                  'Sign Out',
                  style: TextStyle(color: Colors.red),
                ),
              ),
            ],
          ),
    );
  }

  (String, String) _splitName(AuthState auth) {
    if ((auth.firstName ?? '').isNotEmpty && (auth.lastName ?? '').isNotEmpty) {
      return (auth.firstName!, auth.lastName!);
    }
    final parts =
        (auth.userName ?? '')
            .trim()
            .split(RegExp(r'\s+'))
            .where((p) => p.isNotEmpty)
            .toList();
    if (parts.isEmpty) return ('', '');
    if (parts.length == 1) return (parts.first, '');
    return (parts.first, parts.sublist(1).join(' '));
  }
}

class _Header extends StatelessWidget {
  final AuthState auth;

  const _Header({required this.auth});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final label = auth.userName ?? auth.email ?? 'SmartAqua User';
    final parts =
        label.split(RegExp(r'\s+')).where((p) => p.isNotEmpty).toList();
    final initials =
        parts.isEmpty
            ? 'SA'
            : parts.length == 1
            ? parts.first.substring(0, 1).toUpperCase()
            : '${parts.first.substring(0, 1)}${parts.last.substring(0, 1)}'
                .toUpperCase();

    return Container(
      margin: const EdgeInsets.fromLTRB(16, 16, 16, 8),
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.5),
        borderRadius: BorderRadius.circular(20),
      ),
      child: Row(
        children: [
          CircleAvatar(
            radius: 28,
            backgroundColor: theme.colorScheme.primaryContainer,
            child: Text(
              initials,
              style: theme.textTheme.titleMedium?.copyWith(
                fontWeight: FontWeight.bold,
                color: theme.colorScheme.onPrimaryContainer,
              ),
            ),
          ),
          const SizedBox(width: 16),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  label,
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  auth.email ?? 'Profile syncing...',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _StatusBanner extends StatelessWidget {
  final String message;
  final bool isError;
  final String? actionLabel;
  final VoidCallback? onAction;

  const _StatusBanner({
    required this.message,
    required this.isError,
    this.actionLabel,
    this.onAction,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final background =
        isError
            ? theme.colorScheme.errorContainer
            : theme.colorScheme.primaryContainer;
    final foreground =
        isError
            ? theme.colorScheme.onErrorContainer
            : theme.colorScheme.onPrimaryContainer;

    return Container(
      margin: const EdgeInsets.fromLTRB(16, 16, 16, 0),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Row(
        children: [
          Icon(
            isError ? Icons.error_outline : Icons.info_outline,
            color: foreground,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Text(
              message,
              style: TextStyle(color: foreground, fontWeight: FontWeight.w600),
            ),
          ),
          if (actionLabel != null && onAction != null)
            TextButton(onPressed: onAction, child: Text(actionLabel!)),
        ],
      ),
    );
  }
}

class _Section extends StatelessWidget {
  final String title;
  final List<Widget> children;

  const _Section({required this.title, required this.children});

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 8),
          child: Text(
            title,
            style: Theme.of(context).textTheme.titleSmall?.copyWith(
              color: Theme.of(context).colorScheme.primary,
              fontWeight: FontWeight.bold,
            ),
          ),
        ),
        ...children,
        const Divider(),
      ],
    );
  }
}
