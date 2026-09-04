import 'package:equatable/equatable.dart';

class FarmUnit extends Equatable {
  final String id;
  final String name;
  final String type; // 'Earthen Pond', 'Concrete Tank', 'Tarpaulin Tank', 'RAS High-Density', 'Cage'
  final String species;
  final int fishCount;
  final double avgWeightGrams;
  final double targetHarvestWeightGrams;
  final double lengthM;
  final double widthM;
  final double depthM;
  final String? linkedDeviceId;
  final double? manualDO;
  final double? manualTemp;
  final double? manualPh;
  final double? manualTAN;
  final DateTime stockedAt;
  final DateTime? lastSampledAt;

  const FarmUnit({
    required this.id,
    required this.name,
    required this.type,
    required this.species,
    required this.fishCount,
    required this.avgWeightGrams,
    required this.targetHarvestWeightGrams,
    required this.lengthM,
    required this.widthM,
    required this.depthM,
    this.linkedDeviceId,
    this.manualDO,
    this.manualTemp,
    this.manualPh,
    this.manualTAN,
    required this.stockedAt,
    this.lastSampledAt,
  });

  double get volumeM3 => lengthM * widthM * depthM;
  double get surfaceAreaM2 => lengthM * widthM;
  double get totalBiomassKg => (fishCount * avgWeightGrams) / 1000.0;
  double get densityKgM3 => volumeM3 > 0 ? (totalBiomassKg / volumeM3) : 0.0;
  double get growthProgress => (avgWeightGrams / targetHarvestWeightGrams).clamp(0.0, 1.0);
  int get daysInProduction => (DateTime.now().difference(stockedAt).inDays + 1).clamp(1, 365);

  FarmUnit copyWith({
    String? id,
    String? name,
    String? type,
    String? species,
    int? fishCount,
    double? avgWeightGrams,
    double? targetHarvestWeightGrams,
    double? lengthM,
    double? widthM,
    double? depthM,
    String? linkedDeviceId,
    double? manualDO,
    double? manualTemp,
    double? manualPh,
    double? manualTAN,
    DateTime? stockedAt,
    DateTime? lastSampledAt,
  }) {
    return FarmUnit(
      id: id ?? this.id,
      name: name ?? this.name,
      type: type ?? this.type,
      species: species ?? this.species,
      fishCount: fishCount ?? this.fishCount,
      avgWeightGrams: avgWeightGrams ?? this.avgWeightGrams,
      targetHarvestWeightGrams: targetHarvestWeightGrams ?? this.targetHarvestWeightGrams,
      lengthM: lengthM ?? this.lengthM,
      widthM: widthM ?? this.widthM,
      depthM: depthM ?? this.depthM,
      linkedDeviceId: linkedDeviceId ?? this.linkedDeviceId,
      manualDO: manualDO ?? this.manualDO,
      manualTemp: manualTemp ?? this.manualTemp,
      manualPh: manualPh ?? this.manualPh,
      manualTAN: manualTAN ?? this.manualTAN,
      stockedAt: stockedAt ?? this.stockedAt,
      lastSampledAt: lastSampledAt ?? this.lastSampledAt,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'type': type,
      'species': species,
      'fishCount': fishCount,
      'avgWeightGrams': avgWeightGrams,
      'targetHarvestWeightGrams': targetHarvestWeightGrams,
      'lengthM': lengthM,
      'widthM': widthM,
      'depthM': depthM,
      'linkedDeviceId': linkedDeviceId,
      'manualDO': manualDO,
      'manualTemp': manualTemp,
      'manualPh': manualPh,
      'manualTAN': manualTAN,
      'stockedAt': stockedAt.toIso8601String(),
      'lastSampledAt': lastSampledAt?.toIso8601String(),
    };
  }

  factory FarmUnit.fromJson(Map<String, dynamic> json) {
    return FarmUnit(
      id: json['id'] as String,
      name: json['name'] as String? ?? 'Unit',
      type: json['type'] as String? ?? 'Earthen Pond',
      species: json['species'] as String? ?? 'African Catfish (Clarias)',
      fishCount: (json['fishCount'] as num?)?.toInt() ?? 1000,
      avgWeightGrams: (json['avgWeightGrams'] as num?)?.toDouble() ?? 100.0,
      targetHarvestWeightGrams: (json['targetHarvestWeightGrams'] as num?)?.toDouble() ?? 800.0,
      lengthM: (json['lengthM'] as num?)?.toDouble() ?? 12.0,
      widthM: (json['widthM'] as num?)?.toDouble() ?? 8.0,
      depthM: (json['depthM'] as num?)?.toDouble() ?? 1.5,
      linkedDeviceId: json['linkedDeviceId'] as String?,
      manualDO: (json['manualDO'] as num?)?.toDouble(),
      manualTemp: (json['manualTemp'] as num?)?.toDouble(),
      manualPh: (json['manualPh'] as num?)?.toDouble(),
      manualTAN: (json['manualTAN'] as num?)?.toDouble(),
      stockedAt: json['stockedAt'] != null
          ? (DateTime.tryParse(json['stockedAt'] as String) ?? DateTime.now())
          : DateTime.now(),
      lastSampledAt: json['lastSampledAt'] != null
          ? DateTime.tryParse(json['lastSampledAt'] as String)
          : null,
    );
  }

  @override
  List<Object?> get props => [
        id,
        name,
        type,
        species,
        fishCount,
        avgWeightGrams,
        targetHarvestWeightGrams,
        lengthM,
        widthM,
        depthM,
        linkedDeviceId,
        manualDO,
        manualTemp,
        manualPh,
        manualTAN,
        stockedAt,
        lastSampledAt,
      ];
}
