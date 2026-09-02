import 'package:equatable/equatable.dart';

String _stringValue(dynamic value) => value?.toString() ?? '';

class VideoClip extends Equatable {
  final String id;
  final String deviceId;
  final String feedingEventId;
  final String videoUrl;
  final String thumbnailUrl;
  final int durationSeconds;
  final VideoType type;
  final DateTime capturedAt;
  final BoilIndexAnalysis? analysis;

  const VideoClip({
    required this.id,
    required this.deviceId,
    required this.feedingEventId,
    required this.videoUrl,
    required this.thumbnailUrl,
    required this.durationSeconds,
    required this.type,
    required this.capturedAt,
    this.analysis,
  });

  factory VideoClip.fromJson(Map<String, dynamic> json) {
    return VideoClip(
      id: _stringValue(json['id']),
      deviceId: json['device_id'] ?? '',
      feedingEventId: _stringValue(json['feeding_event_id']),
      videoUrl:
          json['video_url'] ?? json['cloud_url'] ?? json['file_path'] ?? '',
      thumbnailUrl: json['thumbnail_url'] ?? '',
      durationSeconds: json['duration_seconds'] ?? 0,
      type: _parseVideoType(json['type']),
      capturedAt: DateTime.parse(
        json['captured_at'] ??
            json['timestamp'] ??
            DateTime.now().toIso8601String(),
      ),
      analysis:
          json['analysis'] != null
              ? BoilIndexAnalysis.fromJson(json['analysis'])
              : null,
    );
  }

  static VideoType _parseVideoType(String? type) {
    switch (type) {
      case 'pre_feed':
        return VideoType.preFeed;
      case 'active_feed':
        return VideoType.activeFeed;
      case 'post_feed':
        return VideoType.postFeed;
      default:
        return VideoType.activeFeed;
    }
  }

  @override
  List<Object?> get props => [id, deviceId, feedingEventId, type, capturedAt];
}

enum VideoType { preFeed, activeFeed, postFeed }

class BoilIndexAnalysis extends Equatable {
  final double boilIndex;
  final double satietyLevel;
  final double pelletCoverage;
  final double strikeRate;
  final double opticalFlowMagnitude;
  final bool feedingComplete;
  final String recommendation;
  final double confidenceScore;
  final DateTime analyzedAt;

  const BoilIndexAnalysis({
    required this.boilIndex,
    required this.satietyLevel,
    required this.pelletCoverage,
    required this.strikeRate,
    required this.opticalFlowMagnitude,
    required this.feedingComplete,
    required this.recommendation,
    required this.confidenceScore,
    required this.analyzedAt,
  });

  factory BoilIndexAnalysis.fromJson(Map<String, dynamic> json) {
    return BoilIndexAnalysis(
      boilIndex:
          (json['boil_index'] ?? json['active_feed_boil_index'] ?? 0)
              .toDouble(),
      satietyLevel:
          (json['satiety_level'] ?? json['surface_activity_level'] ?? 0)
              .toDouble(),
      pelletCoverage: (json['pellet_coverage'] ?? 0).toDouble(),
      strikeRate:
          (json['strike_rate'] ?? json['feeding_efficiency'] ?? 0).toDouble(),
      opticalFlowMagnitude: (json['optical_flow_magnitude'] ?? 0).toDouble(),
      feedingComplete:
          json['feeding_complete'] ?? json['early_cutoff_triggered'] ?? false,
      recommendation: json['recommendation'] ?? '',
      confidenceScore: (json['confidence_score'] ?? 0).toDouble(),
      analyzedAt: DateTime.parse(
        json['analyzed_at'] ??
            json['timestamp'] ??
            DateTime.now().toIso8601String(),
      ),
    );
  }

  String get satietyDescription {
    if (satietyLevel >= 0.8) return 'Full';
    if (satietyLevel >= 0.6) return 'Satisfied';
    if (satietyLevel >= 0.4) return 'Moderate';
    if (satietyLevel >= 0.2) return 'Hungry';
    return 'Very Hungry';
  }

  String get activityDescription {
    if (boilIndex >= 0.8) return 'Very Active';
    if (boilIndex >= 0.6) return 'Active';
    if (boilIndex >= 0.4) return 'Moderate';
    if (boilIndex >= 0.2) return 'Low';
    return 'Inactive';
  }

  @override
  List<Object?> get props => [
    boilIndex,
    satietyLevel,
    pelletCoverage,
    strikeRate,
    confidenceScore,
  ];
}

class FeedingVerification extends Equatable {
  final String feedingEventId;
  final List<VideoClip> clips;
  final BoilIndexAnalysis? preFeedAnalysis;
  final BoilIndexAnalysis? activeFeedAnalysis;
  final BoilIndexAnalysis? postFeedAnalysis;
  final double overallEfficiency;
  final String summary;

  const FeedingVerification({
    required this.feedingEventId,
    required this.clips,
    this.preFeedAnalysis,
    this.activeFeedAnalysis,
    this.postFeedAnalysis,
    required this.overallEfficiency,
    required this.summary,
  });

  factory FeedingVerification.fromJson(Map<String, dynamic> json) {
    return FeedingVerification(
      feedingEventId: json['feeding_event_id'] ?? '',
      clips:
          (json['clips'] as List?)
              ?.map((c) => VideoClip.fromJson(c))
              .toList() ??
          [],
      preFeedAnalysis:
          json['pre_feed_analysis'] != null
              ? BoilIndexAnalysis.fromJson(json['pre_feed_analysis'])
              : null,
      activeFeedAnalysis:
          json['active_feed_analysis'] != null
              ? BoilIndexAnalysis.fromJson(json['active_feed_analysis'])
              : null,
      postFeedAnalysis:
          json['post_feed_analysis'] != null
              ? BoilIndexAnalysis.fromJson(json['post_feed_analysis'])
              : null,
      overallEfficiency: (json['overall_efficiency'] ?? 0).toDouble(),
      summary: json['summary'] ?? '',
    );
  }

  @override
  List<Object?> get props => [feedingEventId, clips, overallEfficiency];
}
