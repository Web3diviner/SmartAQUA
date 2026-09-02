import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:local_auth/local_auth.dart';

import '../../../../core/providers/auth_provider.dart';

class LoginScreen extends ConsumerStatefulWidget {
  const LoginScreen({super.key});

  @override
  ConsumerState<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends ConsumerState<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  bool _obscurePassword = true;

  @override
  void initState() {
    super.initState();
    _tryBiometricLogin();
  }

  Future<void> _tryBiometricLogin() async {
    final authState = ref.read(authStateProvider);
    if (authState.biometricAvailable && authState.biometricEnabled) {
      final success =
          await ref
              .read(authStateProvider.notifier)
              .authenticateWithBiometrics();
      if (success && mounted) {
        context.go('/dashboard');
      }
    }
  }

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    if (_formKey.currentState!.validate()) {
      final success = await ref
          .read(authStateProvider.notifier)
          .login(_emailController.text.trim(), _passwordController.text);

      if (success && mounted) {
        context.go('/dashboard');
      }
    }
  }

  void _showForgotPasswordDialog() {
    final emailController = TextEditingController(text: _emailController.text);
    final codeController = TextEditingController();
    final newPasswordController = TextEditingController();
    int step = 0; // 0: email, 1: code, 2: new password

    showDialog(
      context: context,
      builder:
          (ctx) => StatefulBuilder(
            builder: (ctx, setDialogState) {
              final authState = ref.watch(authStateProvider);

              return AlertDialog(
                title: Text(
                  step == 0
                      ? 'Reset Password'
                      : step == 1
                      ? 'Enter Code'
                      : 'New Password',
                ),
                content: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    if (step == 0) ...[
                      const Text('Enter your email to receive a reset code.'),
                      const SizedBox(height: 16),
                      TextField(
                        controller: emailController,
                        decoration: const InputDecoration(
                          labelText: 'Email',
                          border: OutlineInputBorder(),
                        ),
                        keyboardType: TextInputType.emailAddress,
                      ),
                    ] else if (step == 1) ...[
                      const Text('Enter the 6-digit code sent to your email.'),
                      const SizedBox(height: 16),
                      TextField(
                        controller: codeController,
                        decoration: const InputDecoration(
                          labelText: 'Reset Code',
                          border: OutlineInputBorder(),
                        ),
                        keyboardType: TextInputType.number,
                        maxLength: 6,
                      ),
                    ] else ...[
                      const Text('Enter your new password.'),
                      const SizedBox(height: 16),
                      TextField(
                        controller: newPasswordController,
                        decoration: const InputDecoration(
                          labelText: 'New Password',
                          border: OutlineInputBorder(),
                        ),
                        obscureText: true,
                      ),
                    ],
                    if (authState.error != null)
                      Padding(
                        padding: const EdgeInsets.only(top: 8),
                        child: Text(
                          authState.error!,
                          style: const TextStyle(color: Colors.red),
                        ),
                      ),
                    if (authState.statusMessage != null)
                      Padding(
                        padding: const EdgeInsets.only(top: 8),
                        child: Text(
                          authState.statusMessage!,
                          style: TextStyle(
                            color: Theme.of(context).colorScheme.primary,
                          ),
                        ),
                      ),
                  ],
                ),
                actions: [
                  TextButton(
                    onPressed: () => Navigator.pop(ctx),
                    child: const Text('Cancel'),
                  ),
                  FilledButton(
                    onPressed:
                        authState.isLoading
                            ? null
                            : () async {
                              if (step == 0) {
                                final success = await ref
                                    .read(authStateProvider.notifier)
                                    .requestPasswordReset(
                                      emailController.text.trim(),
                                    );
                                if (success) {
                                  setDialogState(() => step = 1);
                                }
                              } else if (step == 1) {
                                final success = await ref
                                    .read(authStateProvider.notifier)
                                    .verifyResetCode(
                                      emailController.text.trim(),
                                      codeController.text,
                                    );
                                if (success) {
                                  setDialogState(() => step = 2);
                                }
                              } else {
                                final success = await ref
                                    .read(authStateProvider.notifier)
                                    .resetPassword(
                                      emailController.text.trim(),
                                      codeController.text,
                                      newPasswordController.text,
                                    );
                                if (success) {
                                  if (!ctx.mounted || !context.mounted) {
                                    return;
                                  }
                                  Navigator.pop(ctx);
                                  ScaffoldMessenger.of(context).showSnackBar(
                                    const SnackBar(
                                      content: Text(
                                        'Password reset successfully. Please login.',
                                      ),
                                    ),
                                  );
                                }
                              }
                            },
                    child:
                        authState.isLoading
                            ? const SizedBox(
                              height: 16,
                              width: 16,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                            : Text(
                              step == 0
                                  ? 'Send Code'
                                  : step == 1
                                  ? 'Verify'
                                  : 'Reset',
                            ),
                  ),
                ],
              );
            },
          ),
    );
  }

  String _getBiometricLabel(List<BiometricType> types) {
    if (types.contains(BiometricType.face)) return 'Face ID';
    if (types.contains(BiometricType.fingerprint)) return 'Fingerprint';
    return 'Biometric';
  }

  IconData _getBiometricIcon(List<BiometricType> types) {
    if (types.contains(BiometricType.face)) return Icons.face;
    return Icons.fingerprint;
  }

  @override
  Widget build(BuildContext context) {
    final authState = ref.watch(authStateProvider);

    return Scaffold(
      body: SafeArea(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Form(
            key: _formKey,
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                const SizedBox(height: 48),
                Image.asset(
                  'assets/images/logo.png',
                  width: 96,
                  height: 96,
                  fit: BoxFit.contain,
                ),
                const SizedBox(height: 24),
                Text(
                  'Welcome Back',
                  style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 8),
                Text(
                  'Sign in to manage your SmartAqua feeders',
                  style: Theme.of(
                    context,
                  ).textTheme.bodyLarge?.copyWith(color: Colors.grey),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 48),
                TextFormField(
                  controller: _emailController,
                  keyboardType: TextInputType.emailAddress,
                  decoration: const InputDecoration(
                    labelText: 'Email',
                    prefixIcon: Icon(Icons.email_outlined),
                  ),
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Please enter your email';
                    }
                    if (!value.contains('@')) {
                      return 'Please enter a valid email';
                    }
                    return null;
                  },
                ),
                const SizedBox(height: 16),
                TextFormField(
                  controller: _passwordController,
                  obscureText: _obscurePassword,
                  decoration: InputDecoration(
                    labelText: 'Password',
                    prefixIcon: const Icon(Icons.lock_outlined),
                    suffixIcon: IconButton(
                      icon: Icon(
                        _obscurePassword
                            ? Icons.visibility
                            : Icons.visibility_off,
                      ),
                      onPressed:
                          () => setState(
                            () => _obscurePassword = !_obscurePassword,
                          ),
                    ),
                  ),
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Please enter your password';
                    }
                    return null;
                  },
                ),
                Align(
                  alignment: Alignment.centerRight,
                  child: TextButton(
                    onPressed: _showForgotPasswordDialog,
                    child: const Text('Forgot Password?'),
                  ),
                ),
                if (authState.error != null) ...[
                  const SizedBox(height: 8),
                  Text(
                    authState.error!,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                    textAlign: TextAlign.center,
                  ),
                ],
                if (authState.statusMessage != null) ...[
                  const SizedBox(height: 8),
                  Text(
                    authState.statusMessage!,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.primary,
                    ),
                    textAlign: TextAlign.center,
                  ),
                ],
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: authState.isLoading ? null : _login,
                  child:
                      authState.isLoading
                          ? const SizedBox(
                            height: 20,
                            width: 20,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                          : const Text('Sign In'),
                ),
                // Biometric login button
                if (authState.biometricAvailable &&
                    authState.biometricEnabled) ...[
                  const SizedBox(height: 16),
                  FutureBuilder<List<BiometricType>>(
                    future:
                        ref
                            .read(authStateProvider.notifier)
                            .getAvailableBiometrics(),
                    builder: (context, snapshot) {
                      final types = snapshot.data ?? [];
                      return OutlinedButton.icon(
                        onPressed: () async {
                          final success =
                              await ref
                                  .read(authStateProvider.notifier)
                                  .authenticateWithBiometrics();
                          if (success && context.mounted) {
                            context.go('/dashboard');
                          }
                        },
                        icon: Icon(_getBiometricIcon(types)),
                        label: Text(
                          'Sign in with ${_getBiometricLabel(types)}',
                        ),
                      );
                    },
                  ),
                ],
                const SizedBox(height: 16),
                TextButton(
                  onPressed: () => context.go('/register'),
                  child: const Text("Don't have an account? Sign Up"),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
