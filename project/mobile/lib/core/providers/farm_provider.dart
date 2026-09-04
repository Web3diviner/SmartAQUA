import 'dart:convert';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/farm_unit.dart';
import '../services/storage_service.dart';

class FarmUnitsState {
  final List<FarmUnit> units;
  final bool isLoading;

  const FarmUnitsState({
    this.units = const [],
    this.isLoading = false,
  });

  FarmUnitsState copyWith({
    List<FarmUnit>? units,
    bool? isLoading,
  }) {
    return FarmUnitsState(
      units: units ?? this.units,
      isLoading: isLoading ?? this.isLoading,
    );
  }
}

class FarmUnitsNotifier extends StateNotifier<FarmUnitsState> {
  FarmUnitsNotifier() : super(const FarmUnitsState()) {
    loadUnits();
  }

  void loadUnits() {
    final rawJson = StorageService.getFarmUnitsJson();
    if (rawJson != null && rawJson.isNotEmpty) {
      try {
        final List<dynamic> decoded = jsonDecode(rawJson) as List<dynamic>;
        final list = decoded
            .map((e) => FarmUnit.fromJson(e as Map<String, dynamic>))
            .toList();
        state = state.copyWith(units: list);
        return;
      } catch (_) {
        // Fallback to initial defaults if parsing failed
      }
    }

    // If user has not configured any custom units yet, start with empty list
    state = state.copyWith(units: const []);
  }

  Future<void> addUnit(FarmUnit unit) async {
    final updated = [...state.units, unit];
    state = state.copyWith(units: updated);
    await _persistUnits(updated);
  }

  Future<void> updateUnit(FarmUnit updatedUnit) async {
    final updated = state.units.map((u) {
      return u.id == updatedUnit.id ? updatedUnit : u;
    }).toList();
    state = state.copyWith(units: updated);
    await _persistUnits(updated);
  }

  Future<void> deleteUnit(String unitId) async {
    final updated = state.units.where((u) => u.id != unitId).toList();
    state = state.copyWith(units: updated);
    await _persistUnits(updated);
  }

  Future<void> recordSampling(String unitId, double newWeightGrams) async {
    final updated = state.units.map((u) {
      if (u.id == unitId) {
        return u.copyWith(
          avgWeightGrams: newWeightGrams,
          lastSampledAt: DateTime.now(),
        );
      }
      return u;
    }).toList();
    state = state.copyWith(units: updated);
    await _persistUnits(updated);
  }

  Future<void> recordMortality(String unitId, int count) async {
    final updated = state.units.map((u) {
      if (u.id == unitId) {
        final newCount = (u.fishCount - count).clamp(0, 1000000);
        return u.copyWith(fishCount: newCount);
      }
      return u;
    }).toList();
    state = state.copyWith(units: updated);
    await _persistUnits(updated);
  }

  Future<void> updateManualWaterQuality(
    String unitId, {
    double? doMgL,
    double? tempC,
    double? ph,
    double? tanMgL,
  }) async {
    final updated = state.units.map((u) {
      if (u.id == unitId) {
        return u.copyWith(
          manualDO: doMgL ?? u.manualDO,
          manualTemp: tempC ?? u.manualTemp,
          manualPh: ph ?? u.manualPh,
          manualTAN: tanMgL ?? u.manualTAN,
        );
      }
      return u;
    }).toList();
    state = state.copyWith(units: updated);
    await _persistUnits(updated);
  }

  Future<void> _persistUnits(List<FarmUnit> units) async {
    final encoded = jsonEncode(units.map((u) => u.toJson()).toList());
    await StorageService.setFarmUnitsJson(encoded);
  }
}

final farmUnitsProvider =
    StateNotifierProvider<FarmUnitsNotifier, FarmUnitsState>((ref) {
  return FarmUnitsNotifier();
});
