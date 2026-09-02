import 'package:equatable/equatable.dart';

String _stringValue(dynamic value) => value?.toString() ?? '';

double _doubleValue(dynamic value, {double fallback = 0}) {
  if (value is num) return value.toDouble();
  if (value is String) return double.tryParse(value) ?? fallback;
  return fallback;
}

double? _nullableDoubleValue(dynamic value) {
  if (value == null) return null;
  if (value is num) return value.toDouble();
  if (value is String) return double.tryParse(value);
  return null;
}

int _intValue(dynamic value, {int fallback = 0}) {
  if (value is int) return value;
  if (value is num) return value.toInt();
  if (value is String) return int.tryParse(value) ?? fallback;
  return fallback;
}

class FeedingSchedule extends Equatable {
  final String id;
  final String deviceId;
  final String time;
  final double amount;
  final int durationSeconds;
  final List<int> daysOfWeek;
  final bool isEnabled;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  const FeedingSchedule({
    required this.id,
    required this.deviceId,
    required this.time,
    required this.amount,
    this.durationSeconds = 10,
    required this.daysOfWeek,
    required this.isEnabled,
    this.createdAt,
    this.updatedAt,
  });

  factory FeedingSchedule.fromJson(Map<String, dynamic> json) {
    final hour = json['hour'];
    final minute = json['minute'];
    final time =
        json['time'] ??
        (hour != null && minute != null
            ? '${hour.toString().padLeft(2, '0')}:${minute.toString().padLeft(2, '0')}'
            : '08:00');

    return FeedingSchedule(
      id: _stringValue(json['id']),
      deviceId: json['device_id'] ?? '',
      time: time,
      amount: _doubleValue(json['amount'] ?? json['quantity_grams']),
      durationSeconds: _intValue(json['duration_seconds'], fallback: 10),
      daysOfWeek: List<int>.from(json['days_of_week'] ?? [0, 1, 2, 3, 4, 5, 6]),
      isEnabled: json['is_enabled'] ?? json['is_active'] ?? true,
      createdAt:
          json['created_at'] != null
              ? DateTime.parse(json['created_at'])
              : null,
      updatedAt:
          json['updated_at'] != null
              ? DateTime.parse(json['updated_at'])
              : null,
    );
  }

  Map<String, dynamic> toJson() {
    final parts = time.split(':');
    final hour = parts.isNotEmpty ? int.tryParse(parts.first) ?? 8 : 8;
    final minute = parts.length > 1 ? int.tryParse(parts[1]) ?? 0 : 0;

    return {
      if (id.isNotEmpty && int.tryParse(id) != null) 'id': int.parse(id),
      'device_id': deviceId,
      'name': 'Schedule $time',
      'hour': hour,
      'minute': minute,
      'quantity_grams': amount,
      'duration_seconds': durationSeconds,
      'days_of_week': daysOfWeek,
      'is_active': isEnabled,
    };
  }

  FeedingSchedule copyWith({
    String? id,
    String? deviceId,
    String? time,
    double? amount,
    int? durationSeconds,
    List<int>? daysOfWeek,
    bool? isEnabled,
  }) {
    return FeedingSchedule(
      id: id ?? this.id,
      deviceId: deviceId ?? this.deviceId,
      time: time ?? this.time,
      amount: amount ?? this.amount,
      durationSeconds: durationSeconds ?? this.durationSeconds,
      daysOfWeek: daysOfWeek ?? this.daysOfWeek,
      isEnabled: isEnabled ?? this.isEnabled,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }

  String get daysDescription {
    if (daysOfWeek.length == 7) return 'Every day';
    if (daysOfWeek.length == 5 &&
        !daysOfWeek.contains(0) &&
        !daysOfWeek.contains(6)) {
      return 'Weekdays';
    }
    if (daysOfWeek.length == 2 &&
        daysOfWeek.contains(0) &&
        daysOfWeek.contains(6)) {
      return 'Weekends';
    }
    final dayNames = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
    return daysOfWeek.map((d) => dayNames[d]).join(', ');
  }

  @override
  List<Object?> get props => [
    id,
    deviceId,
    time,
    amount,
    durationSeconds,
    daysOfWeek,
    isEnabled,
  ];
}

enum FeedingEventStatus { completed, failed, pending, cancelled }

class FeedingEvent extends Equatable {
  final String id;
  final String deviceId;
  final double amount;
  final double? actualAmount;
  final FeedingEventStatus status;
  final String type;
  final String? errorMessage;
  final DateTime scheduledAt;
  final DateTime? completedAt;
  final double? waterTemperature;
  final double? q10Factor;

  const FeedingEvent({
    required this.id,
    required this.deviceId,
    required this.amount,
    this.actualAmount,
    required this.status,
    required this.type,
    this.errorMessage,
    required this.scheduledAt,
    this.completedAt,
    this.waterTemperature,
    this.q10Factor,
  });

  factory FeedingEvent.fromJson(Map<String, dynamic> json) {
    final requestedAmount = _doubleValue(
      json['amount'] ?? json['quantity_grams'],
    );
    return FeedingEvent(
      id: _stringValue(json['id']),
      deviceId: json['device_id'] ?? '',
      amount: requestedAmount,
      actualAmount: _nullableDoubleValue(
        json['actual_dispensed'] ?? json['actual_amount'] ?? requestedAmount,
      ),
      status: _parseStatus(json['status'] ?? _resultToStatus(json['result'])),
      type: json['type'] ?? json['trigger_type'] ?? 'scheduled',
      errorMessage: json['error_message'],
      scheduledAt: DateTime.parse(
        json['scheduled_at'] ??
            json['timestamp'] ??
            DateTime.now().toIso8601String(),
      ),
      completedAt:
          json['completed_at'] != null
              ? DateTime.parse(json['completed_at'])
              : null,
      waterTemperature: _nullableDoubleValue(
        json['water_temperature'] ?? json['temperature'],
      ),
      q10Factor: _nullableDoubleValue(json['q10_factor']),
    );
  }

  static String _resultToStatus(dynamic result) {
    // FeedingResult firmware enum: 0=SUCCESS 1=PARTIAL 2=TIMEOUT 3=CANCELLED 4=STALL 5=LOW_FEED 6=ERROR
    switch (_intValue(result)) {
      case 0:
        return 'completed';
      case 3:
        return 'cancelled';
      default:
        return 'failed';
    }
  }

  static FeedingEventStatus _parseStatus(String? status) {
    switch (status) {
      case 'completed':
        return FeedingEventStatus.completed;
      case 'failed':
        return FeedingEventStatus.failed;
      case 'pending':
        return FeedingEventStatus.pending;
      case 'cancelled':
        return FeedingEventStatus.cancelled;
      default:
        return FeedingEventStatus.pending;
    }
  }

  @override
  List<Object?> get props => [
    id,
    deviceId,
    amount,
    actualAmount,
    status,
    type,
    scheduledAt,
    waterTemperature,
    q10Factor,
  ];
}

class FeedCalculationRequest {
  final String speciesId;
  final int fishCount;
  final double averageWeight;
  final double waterTemperature;

  FeedCalculationRequest({
    required this.speciesId,
    required this.fishCount,
    required this.averageWeight,
    required this.waterTemperature,
  });

  Map<String, dynamic> toJson() => {
    'populations': [
      {
        'species_id': speciesId,
        'count': fishCount,
        'average_weight': averageWeight,
      },
    ],
    'environmental': {
      'water_temperature': waterTemperature,
      'season': 'summer',
      'weather_condition': 'sunny',
    },
    'use_q10_algorithm': true,
  };
}

class FeedCalculationResult {
  final double recommendedAmount;
  final double biomass;
  final double feedingRate;
  final double q10Factor;
  final double? obmSafetyFactor;
  final String recommendation;
  final int suggestedFeedings;

  FeedCalculationResult({
    required this.recommendedAmount,
    required this.biomass,
    required this.feedingRate,
    required this.q10Factor,
    this.obmSafetyFactor,
    required this.recommendation,
    required this.suggestedFeedings,
  });

  factory FeedCalculationResult.fromJson(Map<String, dynamic> json) {
    final recommendation = json['recommendation'] ?? json;
    final basicRecommendation =
        recommendation['basic_recommendation'] ?? const {};
    final suggestedFeedingsRaw =
        recommendation['final_feeding_frequency'] ?? json['suggested_feedings'];
    final parsedSuggestedFeedings =
        suggestedFeedingsRaw is num
            ? suggestedFeedingsRaw.toInt()
            : int.tryParse(suggestedFeedingsRaw?.toString() ?? '') ?? 2;
    final normalizedSuggestedFeedings =
        parsedSuggestedFeedings <= 0
            ? 2
            : parsedSuggestedFeedings > 2
            ? 2
            : parsedSuggestedFeedings;

    return FeedCalculationResult(
      recommendedAmount: _doubleValue(
        recommendation['final_daily_amount'] ?? json['recommended_amount'],
      ),
      biomass: _doubleValue(
        recommendation['total_biomass_kg'] ??
            basicRecommendation['total_biomass_kg'] ??
            json['biomass'],
      ),
      feedingRate: _doubleValue(
        recommendation['effective_feeding_rate'] ?? json['feeding_rate'],
      ),
      q10Factor: _doubleValue(
        recommendation['q10_recommendation']?['biological_factors']?['q10_factor'] ??
            json['q10_factor'] ??
            1,
      ),
      obmSafetyFactor:
          recommendation['q10_recommendation']?['biological_factors']?['obm_safety_factor']
              ?.toDouble(),
      recommendation:
          recommendation['basic_recommendation']?['environmental_note'] ??
          json['recommendation'] ??
          '',
      suggestedFeedings: normalizedSuggestedFeedings,
    );
  }

  FeedCalculationResult copyWith({
    double? recommendedAmount,
    double? biomass,
    double? feedingRate,
    double? q10Factor,
    double? obmSafetyFactor,
    String? recommendation,
    int? suggestedFeedings,
  }) {
    return FeedCalculationResult(
      recommendedAmount: recommendedAmount ?? this.recommendedAmount,
      biomass: biomass ?? this.biomass,
      feedingRate: feedingRate ?? this.feedingRate,
      q10Factor: q10Factor ?? this.q10Factor,
      obmSafetyFactor: obmSafetyFactor ?? this.obmSafetyFactor,
      recommendation: recommendation ?? this.recommendation,
      suggestedFeedings: suggestedFeedings ?? this.suggestedFeedings,
    );
  }
}

class FishSpecies {
  final String id;
  final String name;
  final double feedingRatePercentage;
  final double q10Coefficient;
  final double referenceTemperature;
  final double optimalTempMin;
  final double optimalTempMax;
  final double fingerlingFeedRate;
  final double juvenileFeedRate;
  final double adultFeedRate;

  FishSpecies({
    required this.id,
    required this.name,
    required this.feedingRatePercentage,
    required this.q10Coefficient,
    required this.referenceTemperature,
    required this.optimalTempMin,
    required this.optimalTempMax,
    required this.fingerlingFeedRate,
    required this.juvenileFeedRate,
    required this.adultFeedRate,
  });

  factory FishSpecies.fromJson(Map<String, dynamic> json) {
    return FishSpecies(
      id: json['id'] ?? '',
      name: json['name'] ?? '',
      feedingRatePercentage: (json['feeding_rate_percentage'] ?? 0).toDouble(),
      q10Coefficient: (json['q10_coefficient'] ?? 2.0).toDouble(),
      referenceTemperature:
          (json['reference_temperature'] ?? json['optimal_temp_min'] ?? 25)
              .toDouble(),
      optimalTempMin: (json['optimal_temp_min'] ?? 24).toDouble(),
      optimalTempMax: (json['optimal_temp_max'] ?? 30).toDouble(),
      fingerlingFeedRate:
          (json['fingerling_feed_rate'] ?? json['feeding_rate_percentage'] ?? 8)
              .toDouble(),
      juvenileFeedRate:
          (json['juvenile_feed_rate'] ?? json['feeding_rate_percentage'] ?? 4)
              .toDouble(),
      adultFeedRate:
          (json['adult_feed_rate'] ?? json['feeding_rate_percentage'] ?? 1.5)
              .toDouble(),
    );
  }
}
