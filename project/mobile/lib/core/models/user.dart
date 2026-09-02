import 'package:equatable/equatable.dart';

class User extends Equatable {
  final String id;
  final String name;
  final String email;
  final String? phone;
  final String? avatarUrl;
  final DateTime createdAt;
  final DateTime? updatedAt;
  final UserPreferences preferences;

  const User({
    required this.id,
    required this.name,
    required this.email,
    this.phone,
    this.avatarUrl,
    required this.createdAt,
    this.updatedAt,
    required this.preferences,
  });

  factory User.fromJson(Map<String, dynamic> json) {
    return User(
      id: json['id'] ?? '',
      name:
          json['name'] ??
          [
            json['first_name'],
            json['last_name'],
          ].where((v) => v != null && v.toString().trim().isNotEmpty).join(' '),
      email: json['email'] ?? '',
      phone: json['phone'],
      avatarUrl: json['avatar_url'],
      createdAt: DateTime.parse(
        json['created_at'] ?? DateTime.now().toIso8601String(),
      ),
      updatedAt:
          json['updated_at'] != null
              ? DateTime.parse(json['updated_at'])
              : null,
      preferences: UserPreferences.fromJson(json['preferences'] ?? {}),
    );
  }

  Map<String, dynamic> toJson() => {
    'id': id,
    'name': name,
    'email': email,
    'phone': phone,
    'avatar_url': avatarUrl,
    'created_at': createdAt.toIso8601String(),
    'updated_at': updatedAt?.toIso8601String(),
    'preferences': preferences.toJson(),
  };

  User copyWith({
    String? id,
    String? name,
    String? email,
    String? phone,
    String? avatarUrl,
    DateTime? createdAt,
    DateTime? updatedAt,
    UserPreferences? preferences,
  }) {
    return User(
      id: id ?? this.id,
      name: name ?? this.name,
      email: email ?? this.email,
      phone: phone ?? this.phone,
      avatarUrl: avatarUrl ?? this.avatarUrl,
      createdAt: createdAt ?? this.createdAt,
      updatedAt: updatedAt ?? this.updatedAt,
      preferences: preferences ?? this.preferences,
    );
  }

  @override
  List<Object?> get props => [
    id,
    name,
    email,
    phone,
    avatarUrl,
    createdAt,
    preferences,
  ];
}

class UserPreferences extends Equatable {
  final String temperatureUnit;
  final String weightUnit;
  final bool notificationsEnabled;
  final bool emailNotifications;
  final bool pushNotifications;
  final String language;
  final String timezone;

  const UserPreferences({
    this.temperatureUnit = 'celsius',
    this.weightUnit = 'grams',
    this.notificationsEnabled = true,
    this.emailNotifications = true,
    this.pushNotifications = true,
    this.language = 'en',
    this.timezone = 'UTC',
  });

  factory UserPreferences.fromJson(Map<String, dynamic> json) {
    return UserPreferences(
      temperatureUnit: json['temperature_unit'] ?? 'celsius',
      weightUnit: json['weight_unit'] ?? 'grams',
      notificationsEnabled: json['notifications_enabled'] ?? true,
      emailNotifications: json['email_notifications'] ?? true,
      pushNotifications: json['push_notifications'] ?? true,
      language: json['language'] ?? 'en',
      timezone: json['timezone'] ?? 'UTC',
    );
  }

  Map<String, dynamic> toJson() => {
    'temperature_unit': temperatureUnit,
    'weight_unit': weightUnit,
    'notifications_enabled': notificationsEnabled,
    'email_notifications': emailNotifications,
    'push_notifications': pushNotifications,
    'language': language,
    'timezone': timezone,
  };

  UserPreferences copyWith({
    String? temperatureUnit,
    String? weightUnit,
    bool? notificationsEnabled,
    bool? emailNotifications,
    bool? pushNotifications,
    String? language,
    String? timezone,
  }) {
    return UserPreferences(
      temperatureUnit: temperatureUnit ?? this.temperatureUnit,
      weightUnit: weightUnit ?? this.weightUnit,
      notificationsEnabled: notificationsEnabled ?? this.notificationsEnabled,
      emailNotifications: emailNotifications ?? this.emailNotifications,
      pushNotifications: pushNotifications ?? this.pushNotifications,
      language: language ?? this.language,
      timezone: timezone ?? this.timezone,
    );
  }

  @override
  List<Object?> get props => [
    temperatureUnit,
    weightUnit,
    notificationsEnabled,
    emailNotifications,
    pushNotifications,
    language,
    timezone,
  ];
}
