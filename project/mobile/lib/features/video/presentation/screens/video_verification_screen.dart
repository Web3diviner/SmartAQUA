import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:video_player/video_player.dart';
import 'package:chewie/chewie.dart';

import '../../../../core/models/device.dart';
import '../../../../core/models/video_verification.dart';
import '../../../../core/config/env_config.dart';
import '../../../../core/providers/device_provider.dart';
import '../../../../core/providers/video_provider.dart';
import '../../../../core/services/storage_service.dart';

class VideoVerificationScreen extends ConsumerStatefulWidget {
  final String? feedingEventId;

  const VideoVerificationScreen({super.key, this.feedingEventId});

  @override
  ConsumerState<VideoVerificationScreen> createState() =>
      _VideoVerificationScreenState();
}

class _VideoVerificationScreenState
    extends ConsumerState<VideoVerificationScreen> {
  String? _selectedDeviceId;

  @override
  void initState() {
    super.initState();
    Future.microtask(_loadData);
  }

  Future<void> _loadData() async {
    if (widget.feedingEventId != null) {
      await ref
          .read(videoVerificationProvider.notifier)
          .loadVerification(widget.feedingEventId!);
    } else {
      await ref.read(deviceListProvider.notifier).loadDevices();
      final devices = ref.read(devicesProvider);
      if (devices.isNotEmpty && _selectedDeviceId == null) {
        _selectedDeviceId = devices.first.id;
        await ref
            .read(videoVerificationProvider.notifier)
            .loadRecentClips(_selectedDeviceId!);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final videoState = ref.watch(videoVerificationProvider);
    final deviceState = ref.watch(deviceListProvider);

    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.feedingEventId != null
              ? 'Feeding Verification'
              : 'Video Clips',
        ),
        actions: [
          if (_selectedDeviceId != null)
            IconButton(
              icon: const Icon(Icons.videocam),
              onPressed: () => _requestCapture(),
              tooltip: 'Request Video Capture',
            ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _loadData,
        child:
            videoState.isLoading
                ? const Center(child: CircularProgressIndicator())
                : widget.feedingEventId != null
                ? _buildVerificationView(videoState.verification)
                : _buildClipsView(videoState.recentClips, deviceState.devices),
      ),
    );
  }

  Widget _buildVerificationView(FeedingVerification? verification) {
    if (verification == null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Icon(Icons.videocam_off, size: 64, color: Colors.grey[400]),
            const SizedBox(height: 16),
            Text(
              'No verification data available',
              style: TextStyle(color: Colors.grey[600]),
            ),
          ],
        ),
      );
    }

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          // Overall efficiency card
          Card(
            color: Theme.of(context).colorScheme.primaryContainer,
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                children: [
                  Text(
                    'Feeding Efficiency',
                    style: Theme.of(context).textTheme.titleMedium,
                  ),
                  const SizedBox(height: 8),
                  Text(
                    '${(verification.overallEfficiency * 100).toInt()}%',
                    style: Theme.of(context).textTheme.displayMedium?.copyWith(
                      fontWeight: FontWeight.bold,
                      color: _getEfficiencyColor(
                        verification.overallEfficiency,
                      ),
                    ),
                  ),
                  const SizedBox(height: 8),
                  Text(
                    verification.summary,
                    textAlign: TextAlign.center,
                    style: TextStyle(color: Colors.grey[700]),
                  ),
                ],
              ),
            ),
          ),
          const SizedBox(height: 24),

          // Analysis phases
          Text(
            'Analysis Phases',
            style: Theme.of(
              context,
            ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 12),

          if (verification.preFeedAnalysis != null)
            _AnalysisCard(
              title: 'Pre-Feed',
              icon: Icons.play_arrow,
              analysis: verification.preFeedAnalysis!,
              clip:
                  verification.clips
                      .where((c) => c.type == VideoType.preFeed)
                      .firstOrNull,
            ),

          if (verification.activeFeedAnalysis != null)
            _AnalysisCard(
              title: 'Active Feeding',
              icon: Icons.restaurant,
              analysis: verification.activeFeedAnalysis!,
              clip:
                  verification.clips
                      .where((c) => c.type == VideoType.activeFeed)
                      .firstOrNull,
            ),

          if (verification.postFeedAnalysis != null)
            _AnalysisCard(
              title: 'Post-Feed',
              icon: Icons.stop,
              analysis: verification.postFeedAnalysis!,
              clip:
                  verification.clips
                      .where((c) => c.type == VideoType.postFeed)
                      .firstOrNull,
            ),

          const SizedBox(height: 24),

          // Video clips
          if (verification.clips.isNotEmpty) ...[
            Text(
              'Video Clips',
              style: Theme.of(
                context,
              ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 12),
            SizedBox(
              height: 120,
              child: ListView.separated(
                scrollDirection: Axis.horizontal,
                itemCount: verification.clips.length,
                separatorBuilder: (_, _) => const SizedBox(width: 12),
                itemBuilder:
                    (context, index) =>
                        _VideoThumbnail(clip: verification.clips[index]),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildClipsView(List<VideoClip> clips, List<Device> devices) {
    return CustomScrollView(
      slivers: [
        // Device selector
        SliverToBoxAdapter(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Card(
              child: ListTile(
                leading: const Icon(Icons.router),
                title: Text(_getSelectedDeviceName(devices)),
                trailing: const Icon(Icons.arrow_drop_down),
                onTap: () => _showDeviceSelector(context, devices),
              ),
            ),
          ),
        ),

        // Clips grid
        if (clips.isEmpty)
          SliverFillRemaining(
            child: Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  Icon(Icons.video_library, size: 64, color: Colors.grey[400]),
                  const SizedBox(height: 16),
                  Text(
                    'No video clips yet',
                    style: TextStyle(color: Colors.grey[600]),
                  ),
                  const SizedBox(height: 8),
                  ElevatedButton.icon(
                    onPressed: _requestCapture,
                    icon: const Icon(Icons.videocam),
                    label: const Text('Request Capture'),
                  ),
                ],
              ),
            ),
          )
        else
          SliverPadding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            sliver: SliverGrid(
              gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
                crossAxisCount: 2,
                mainAxisSpacing: 12,
                crossAxisSpacing: 12,
                childAspectRatio: 1.2,
              ),
              delegate: SliverChildBuilderDelegate(
                (context, index) => _VideoClipCard(clip: clips[index]),
                childCount: clips.length,
              ),
            ),
          ),
      ],
    );
  }

  String _getSelectedDeviceName(List<Device> devices) {
    if (_selectedDeviceId == null) return 'Select a device';
    try {
      return devices.firstWhere((d) => d.id == _selectedDeviceId).name;
    } catch (_) {
      return 'Unknown device';
    }
  }

  void _showDeviceSelector(BuildContext context, List<Device> devices) {
    showModalBottomSheet(
      context: context,
      builder:
          (ctx) => ListView(
            shrinkWrap: true,
            children: [
              const Padding(
                padding: EdgeInsets.all(16),
                child: Text(
                  'Select Device',
                  style: TextStyle(fontWeight: FontWeight.bold, fontSize: 18),
                ),
              ),
              ...devices.map(
                (device) => ListTile(
                  leading: Icon(
                    Icons.router,
                    color: device.isOnline ? Colors.green : Colors.grey,
                  ),
                  title: Text(device.name),
                  trailing:
                      _selectedDeviceId == device.id
                          ? const Icon(Icons.check, color: Colors.green)
                          : null,
                  onTap: () {
                    setState(() => _selectedDeviceId = device.id);
                    ref
                        .read(videoVerificationProvider.notifier)
                        .loadRecentClips(device.id);
                    Navigator.pop(ctx);
                  },
                ),
              ),
            ],
          ),
    );
  }

  Future<void> _requestCapture() async {
    if (_selectedDeviceId == null) return;

    final success = await ref
        .read(videoVerificationProvider.notifier)
        .requestVideoCapture(_selectedDeviceId!);
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            success ? 'Video capture requested' : 'Failed to request capture',
          ),
          backgroundColor: success ? Colors.green : Colors.red,
        ),
      );
    }
  }

  Color _getEfficiencyColor(double efficiency) {
    if (efficiency >= 0.8) return Colors.green;
    if (efficiency >= 0.6) return Colors.orange;
    return Colors.red;
  }
}

class _AnalysisCard extends StatelessWidget {
  final String title;
  final IconData icon;
  final BoilIndexAnalysis analysis;
  final VideoClip? clip;

  const _AnalysisCard({
    required this.title,
    required this.icon,
    required this.analysis,
    this.clip,
  });

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon, color: Theme.of(context).colorScheme.primary),
                const SizedBox(width: 8),
                Text(
                  title,
                  style: Theme.of(context).textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                  ),
                ),
                const Spacer(),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: _getConfidenceColor(
                      analysis.confidenceScore,
                    ).withValues(alpha: 0.2),
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Text(
                    '${(analysis.confidenceScore * 100).toInt()}% confidence',
                    style: TextStyle(
                      color: _getConfidenceColor(analysis.confidenceScore),
                      fontSize: 12,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Row(
              children: [
                Expanded(
                  child: _MetricItem(
                    label: 'Boil Index',
                    value: analysis.activityDescription,
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'Satiety',
                    value: analysis.satietyDescription,
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'Pellet Coverage',
                    value: '${(analysis.pelletCoverage * 100).toInt()}%',
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: _MetricItem(
                    label: 'Strike Rate',
                    value: '${analysis.strikeRate.toStringAsFixed(1)}/s',
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'Optical Flow',
                    value: analysis.opticalFlowMagnitude.toStringAsFixed(2),
                  ),
                ),
                Expanded(
                  child: _MetricItem(
                    label: 'Status',
                    value:
                        analysis.feedingComplete ? 'Complete' : 'In Progress',
                    valueColor:
                        analysis.feedingComplete ? Colors.green : Colors.orange,
                  ),
                ),
              ],
            ),
            if (analysis.recommendation.isNotEmpty) ...[
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: Colors.blue.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Row(
                  children: [
                    const Icon(
                      Icons.lightbulb_outline,
                      size: 16,
                      color: Colors.blue,
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        analysis.recommendation,
                        style: const TextStyle(fontSize: 12),
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Color _getConfidenceColor(double confidence) {
    if (confidence >= 0.8) return Colors.green;
    if (confidence >= 0.6) return Colors.orange;
    return Colors.red;
  }
}

class _MetricItem extends StatelessWidget {
  final String label;
  final String value;
  final Color? valueColor;

  const _MetricItem({
    required this.label,
    required this.value,
    this.valueColor,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: TextStyle(color: Colors.grey[600], fontSize: 12)),
        Text(
          value,
          style: TextStyle(fontWeight: FontWeight.bold, color: valueColor),
        ),
      ],
    );
  }
}

class _VideoThumbnail extends StatelessWidget {
  final VideoClip clip;

  const _VideoThumbnail({required this.clip});

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: () => _playVideo(context),
      child: Container(
        width: 160,
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(8),
          color: Colors.grey[300],
        ),
        child: Stack(
          children: [
            if (clip.thumbnailUrl.isNotEmpty)
              ClipRRect(
                borderRadius: BorderRadius.circular(8),
                child: Image.network(
                  clip.thumbnailUrl,
                  fit: BoxFit.cover,
                  width: 160,
                  height: 120,
                  errorBuilder: (_, _, _) => _buildPlaceholder(),
                ),
              )
            else
              _buildPlaceholder(),
            Positioned.fill(
              child: Container(
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(8),
                  gradient: LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [
                      Colors.transparent,
                      Colors.black.withValues(alpha: 0.7),
                    ],
                  ),
                ),
              ),
            ),
            const Positioned.fill(
              child: Center(
                child: Icon(
                  Icons.play_circle_fill,
                  color: Colors.white,
                  size: 40,
                ),
              ),
            ),
            Positioned(
              bottom: 8,
              left: 8,
              right: 8,
              child: Text(
                _getTypeLabel(clip.type),
                style: const TextStyle(color: Colors.white, fontSize: 12),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildPlaceholder() {
    return Container(
      width: 160,
      height: 120,
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(8),
        color: Colors.grey[400],
      ),
      child: const Icon(Icons.videocam, color: Colors.white, size: 40),
    );
  }

  String _getTypeLabel(VideoType type) {
    switch (type) {
      case VideoType.preFeed:
        return 'Pre-Feed';
      case VideoType.activeFeed:
        return 'Active Feeding';
      case VideoType.postFeed:
        return 'Post-Feed';
    }
  }

  void _playVideo(BuildContext context) {
    if (clip.videoUrl.isEmpty) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Video not available')));
      return;
    }

    Navigator.of(context).push(
      MaterialPageRoute(
        builder:
            (ctx) => _VideoPlayerScreen(
              videoUrl: clip.videoUrl,
              title: _getTypeLabel(clip.type),
            ),
      ),
    );
  }
}

class _VideoClipCard extends StatelessWidget {
  final VideoClip clip;

  const _VideoClipCard({required this.clip});

  @override
  Widget build(BuildContext context) {
    return Card(
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: () => _showClipDetails(context),
        child: Stack(
          children: [
            if (clip.thumbnailUrl.isNotEmpty)
              Image.network(
                clip.thumbnailUrl,
                fit: BoxFit.cover,
                width: double.infinity,
                height: double.infinity,
                errorBuilder: (_, _, _) => _buildPlaceholder(),
              )
            else
              _buildPlaceholder(),
            Positioned.fill(
              child: Container(
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    colors: [
                      Colors.transparent,
                      Colors.black.withValues(alpha: 0.8),
                    ],
                  ),
                ),
              ),
            ),
            const Positioned.fill(
              child: Center(
                child: Icon(
                  Icons.play_circle_fill,
                  color: Colors.white,
                  size: 48,
                ),
              ),
            ),
            Positioned(
              bottom: 8,
              left: 8,
              right: 8,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    _getTypeLabel(clip.type),
                    style: const TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                  Text(
                    DateFormat('MMM d, h:mm a').format(clip.capturedAt),
                    style: TextStyle(
                      color: Colors.white.withValues(alpha: 0.8),
                      fontSize: 12,
                    ),
                  ),
                ],
              ),
            ),
            Positioned(
              top: 8,
              right: 8,
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: Colors.black.withValues(alpha: 0.6),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(
                  '${clip.durationSeconds}s',
                  style: const TextStyle(color: Colors.white, fontSize: 12),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildPlaceholder() {
    return Container(
      color: Colors.grey[400],
      child: const Center(
        child: Icon(Icons.videocam, color: Colors.white, size: 40),
      ),
    );
  }

  String _getTypeLabel(VideoType type) {
    switch (type) {
      case VideoType.preFeed:
        return 'Pre-Feed';
      case VideoType.activeFeed:
        return 'Active Feeding';
      case VideoType.postFeed:
        return 'Post-Feed';
    }
  }

  void _showClipDetails(BuildContext context) {
    showModalBottomSheet(
      context: context,
      builder:
          (ctx) => Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  _getTypeLabel(clip.type),
                  style: const TextStyle(
                    fontWeight: FontWeight.bold,
                    fontSize: 18,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Captured: ${DateFormat('MMM d, yyyy h:mm a').format(clip.capturedAt)}',
                ),
                Text('Duration: ${clip.durationSeconds} seconds'),
                if (clip.analysis != null) ...[
                  const SizedBox(height: 16),
                  const Text(
                    'Analysis',
                    style: TextStyle(fontWeight: FontWeight.bold),
                  ),
                  Text('Boil Index: ${clip.analysis!.activityDescription}'),
                  Text('Satiety: ${clip.analysis!.satietyDescription}'),
                  Text(
                    'Pellet Coverage: ${(clip.analysis!.pelletCoverage * 100).toInt()}%',
                  ),
                ],
                const SizedBox(height: 16),
                SizedBox(
                  width: double.infinity,
                  child: FilledButton.icon(
                    onPressed: () {
                      Navigator.pop(ctx);
                      if (clip.videoUrl.isNotEmpty) {
                        Navigator.of(context).push(
                          MaterialPageRoute(
                            builder:
                                (_) => _VideoPlayerScreen(
                                  videoUrl: clip.videoUrl,
                                  title: _getTypeLabel(clip.type),
                                ),
                          ),
                        );
                      } else {
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(content: Text('Video not available')),
                        );
                      }
                    },
                    icon: const Icon(Icons.play_arrow),
                    label: const Text('Play Video'),
                  ),
                ),
              ],
            ),
          ),
    );
  }
}

/// Full-screen video player
class _VideoPlayerScreen extends StatefulWidget {
  final String videoUrl;
  final String title;

  const _VideoPlayerScreen({required this.videoUrl, required this.title});

  @override
  State<_VideoPlayerScreen> createState() => _VideoPlayerScreenState();
}

class _VideoPlayerScreenState extends State<_VideoPlayerScreen> {
  late VideoPlayerController _videoController;
  ChewieController? _chewieController;
  bool _isLoading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _initializePlayer();
  }

  Future<void> _initializePlayer() async {
    try {
      final uri = Uri.parse(widget.videoUrl);
      final headers = <String, String>{};

      if (uri.host == EnvConfig.apiDomain &&
          uri.path.contains('/api/v1/vision/clips/')) {
        final token = await StorageService.getAccessToken();
        if (token != null && token.isNotEmpty) {
          headers['Authorization'] = 'Bearer $token';
        }
      }

      _videoController = VideoPlayerController.networkUrl(
        uri,
        httpHeaders: headers,
      );
      await _videoController.initialize();

      _chewieController = ChewieController(
        videoPlayerController: _videoController,
        autoPlay: true,
        looping: false,
        aspectRatio: _videoController.value.aspectRatio,
        allowFullScreen: true,
        allowMuting: true,
        showControls: true,
        materialProgressColors: ChewieProgressColors(
          playedColor: Colors.blue,
          handleColor: Colors.blue,
          backgroundColor: Colors.grey,
          bufferedColor: Colors.lightBlue,
        ),
        placeholder: Container(
          color: Colors.black,
          child: const Center(child: CircularProgressIndicator()),
        ),
        errorBuilder: (context, errorMessage) {
          return Center(
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                const Icon(Icons.error, color: Colors.red, size: 48),
                const SizedBox(height: 8),
                Text(errorMessage, style: const TextStyle(color: Colors.white)),
              ],
            ),
          );
        },
      );

      setState(() => _isLoading = false);
    } catch (e) {
      setState(() {
        _isLoading = false;
        _error = 'Failed to load video: $e';
      });
    }
  }

  @override
  void dispose() {
    _videoController.dispose();
    _chewieController?.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      appBar: AppBar(
        backgroundColor: Colors.black,
        foregroundColor: Colors.white,
        title: Text(widget.title),
      ),
      body: Center(
        child:
            _isLoading
                ? const CircularProgressIndicator()
                : _error != null
                ? Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const Icon(Icons.error, color: Colors.red, size: 64),
                    const SizedBox(height: 16),
                    Text(_error!, style: const TextStyle(color: Colors.white)),
                    const SizedBox(height: 16),
                    ElevatedButton(
                      onPressed: () {
                        setState(() {
                          _isLoading = true;
                          _error = null;
                        });
                        _initializePlayer();
                      },
                      child: const Text('Retry'),
                    ),
                  ],
                )
                : _chewieController != null
                ? Chewie(controller: _chewieController!)
                : const Text(
                  'Unable to play video',
                  style: TextStyle(color: Colors.white),
                ),
      ),
    );
  }
}
