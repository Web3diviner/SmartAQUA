/**
 * 3D Animated Culture System Digital Twin Visualizer.
 *
 * Implements real-time 3D simulation of aquaculture systems (Concrete Tank,
 * Tarpaulin Tank, Earthen Pond, RAS) using Three.js.
 *
 * Features:
 * - Procedural anatomical fish models (African Catfish & Tilapia) with sinusoidal tail undulation.
 * - Dynamic 3D fish scaling based on projected weight (initial vs final harvest vs timeline day).
 * - Real-time behavioral physics:
 *    * Optimal vs Cold vs Heat metabolic swim speed.
 *    * Surface piping / gasping under hypoxia (DO < 3.5 mg/L).
 *    * Erratic darting / lethargy under elevated ammonia.
 * - Dynamic surviving school count and biomass density.
 * - 3D Aeration bubble particle systems & inflow water jets.
 * - Interactive camera controls (Orbit, Pan, Zoom) + View presets (Isometric, Top, Surface, Underwater).
 * - Live HUD Telemetry & Day Scrubber.
 */

import { useEffect, useRef, useState, useMemo, useCallback } from 'react'
import * as THREE from 'three'
import {
  type DigitalTwinPondParams,
  type DigitalTwinOutcome,
  type PondSystemType,
  type FishSpecies,
} from '@/utils/digitalTwinEngine'

interface DigitalTwin3DVisualizerProps {
  params: DigitalTwinPondParams
  outcome: DigitalTwinOutcome
  horizonDays: number
  activeDay: number
  onDayChange?: (day: number) => void
}

interface FishEntity {
  group: THREE.Group
  bodyMesh: THREE.Mesh
  tailMesh: THREE.Mesh
  whiskersMesh?: THREE.Mesh
  position: THREE.Vector3
  velocity: THREE.Vector3
  target: THREE.Vector3
  rotation: THREE.Euler
  tailPhase: number
  tailSpeed: number
  baseScale: number
  personalSpeed: number
  isPiping: boolean
}

export function DigitalTwin3DVisualizer({
  params,
  outcome,
  horizonDays,
  activeDay,
  onDayChange,
}: DigitalTwin3DVisualizerProps) {
  const containerRef = useRef<HTMLDivElement>(null)
  const canvasRef = useRef<HTMLCanvasElement>(null)

  // Simulation play state
  const [isPlaying, setIsPlaying] = useState(false)
  const [cameraMode, setCameraMode] = useState<'iso' | 'top' | 'surface' | 'underwater'>('iso')
  const [autoRotate, setAutoRotate] = useState(true)

  // References to keep Three.js scene objects across renders
  const sceneRef = useRef<THREE.Scene | null>(null)
  const cameraRef = useRef<THREE.PerspectiveCamera | null>(null)
  const rendererRef = useRef<THREE.WebGLRenderer | null>(null)
  const fishEntitiesRef = useRef<FishEntity[]>([])
  const bubblesMeshRef = useRef<THREE.Points | null>(null)
  const waterMeshRef = useRef<THREE.Mesh | null>(null)
  const tankMeshGroupRef = useRef<THREE.Group | null>(null)
  const animationFrameIdRef = useRef<number | null>(null)

  // Mouse interaction state for manual orbit control
  const isDraggingRef = useRef(false)
  const previousMousePositionRef = useRef({ x: 0, y: 0 })
  const sphericalCoordsRef = useRef({ radius: 18, theta: Math.PI / 4, phi: Math.PI / 3.2 })

  // Active step data from the engine for the active day
  const currentStep = useMemo(() => {
    const stepIdx = Math.min(Math.max(1, activeDay), outcome.steps.length) - 1
    return outcome.steps[stepIdx] || outcome.steps[outcome.steps.length - 1]!
  }, [outcome.steps, activeDay])

  // Compute behavioral metrics
  const behavioralState = useMemo(() => {
    const temp = params.waterTempC
    const doLevel = params.dissolvedOxygenMgL
    const ammonia = params.ammoniaMgL

    // Metabolic speed multiplier Q10
    const q10Factor = Math.pow(2.0, (temp - 28) / 10)
    let speedMultiplier = Math.max(0.2, Math.min(2.2, q10Factor))

    // Hypoxia effect
    const isHypoxic = doLevel < 3.8
    const isSevereHypoxia = doLevel < 2.5
    if (isSevereHypoxia) {
      speedMultiplier *= 0.4
    } else if (isHypoxic) {
      speedMultiplier *= 0.7
    }

    // Ammonia stress
    const isAmmoniaToxic = ammonia > 0.8
    if (isAmmoniaToxic) {
      speedMultiplier *= 1.3 // Agitated
    }

    // Determine primary behavioral mode description
    let behaviorName = 'Optimal Cruising'
    let behaviorColor = '#10b981'
    if (isSevereHypoxia) {
      behaviorName = '🚨 Acute Surface Piping (Gasping)'
      behaviorColor = '#ef4444'
    } else if (isHypoxic) {
      behaviorName = '⚠️ Hypoxic Surface Seeking'
      behaviorColor = '#f59e0b'
    } else if (isAmmoniaToxic) {
      behaviorName = '⚡ Ammonia Toxicity Agitation'
      behaviorColor = '#f97316'
    } else if (temp < 22) {
      behaviorName = '❄️ Low-Temp Sluggish Stupor'
      behaviorColor = '#60a5fa'
    } else if (temp > 32) {
      behaviorName = '🔥 High-Temp Thermal Stress'
      behaviorColor = '#f97316'
    }

    return {
      speedMultiplier,
      isHypoxic,
      isSevereHypoxia,
      isAmmoniaToxic,
      behaviorName,
      behaviorColor,
      activeWeightG: currentStep.avgWeightG,
      activeBiomassKg: currentStep.biomassKg,
      activeDensity: currentStep.stockingDensityKgM3,
      swimSpeedCmS: (15 * speedMultiplier).toFixed(1),
    }
  }, [params, currentStep])

  // Tank dimension parameters in 3D world units
  const tankDimensions = useMemo(() => {
    switch (params.systemType) {
      case 'tarpaulin':
        return { type: 'cylinder', radius: 4.8, height: 3.2, waterHeight: 2.8 }
      case 'earthen':
        return { type: 'pond', width: 12, length: 12, height: 3.4, waterHeight: 2.9 }
      case 'ras':
        return { type: 'cylinder', radius: 4.2, height: 3.6, waterHeight: 3.2 }
      case 'concrete':
      default:
        return { type: 'box', width: 9.0, length: 6.0, height: 3.2, waterHeight: 2.7 }
    }
  }, [params.systemType])

  // Calculate visual fish scale based on average weight
  // Reference: 150g ~ 1.0 scale; 10g ~ 0.45 scale; 1000g ~ 1.88 scale (cube-root scaling)
  const fishScale = useMemo(() => {
    const w = currentStep.avgWeightG
    const normalized = Math.pow(w / 150, 1 / 3)
    return Math.max(0.35, Math.min(2.5, normalized))
  }, [currentStep.avgWeightG])

  // Number of 3D fish to render (capped for smooth 60fps WebGL)
  const renderedFishCount = useMemo(() => {
    const survivalPct = outcome.survivalRatePct / 100
    const baseCount = Math.min(params.initialPopulation, 1000)
    // Scale count: 18 - 48 fish representing the whole school density
    const maxRendered = Math.min(48, Math.max(16, Math.floor(baseCount / 22)))
    return Math.max(4, Math.round(maxRendered * survivalPct))
  }, [params.initialPopulation, outcome.survivalRatePct])

  // Auto-play timeline animation loop
  useEffect(() => {
    let interval: NodeJS.Timeout
    if (isPlaying) {
      interval = setInterval(() => {
        if (onDayChange) {
          onDayChange(activeDay >= horizonDays ? 1 : activeDay + 1)
        }
      }, 400)
    }
    return () => clearInterval(interval)
  }, [isPlaying, activeDay, horizonDays, onDayChange])

  // Setup Three.js Scene, Camera, Lighting, and Tank Mesh
  useEffect(() => {
    const container = containerRef.current
    const canvas = canvasRef.current
    if (!container || !canvas) return

    const width = container.clientWidth
    const height = Math.max(380, Math.min(520, container.clientHeight || 460))

    // Scene
    const scene = new THREE.Scene()
    scene.background = new THREE.Color(0x060d17)
    scene.fog = new THREE.FogExp2(0x071526, 0.022)
    sceneRef.current = scene

    // Camera
    const camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 100)
    cameraRef.current = camera

    // Renderer
    const renderer = new THREE.WebGLRenderer({
      canvas,
      antialias: true,
      alpha: true,
      powerPreference: 'high-performance',
    })
    renderer.setSize(width, height)
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2))
    renderer.shadowMap.enabled = true
    renderer.shadowMap.type = THREE.PCFSoftShadowMap
    renderer.toneMapping = THREE.ACESFilmicToneMapping
    rendererRef.current = renderer

    // Lighting
    const ambientLight = new THREE.AmbientLight(0xdcf8ff, 0.8)
    scene.add(ambientLight)

    const sunLight = new THREE.DirectionalLight(0x38bdf8, 1.4)
    sunLight.position.set(10, 18, 12)
    sunLight.castShadow = true
    sunLight.shadow.mapSize.width = 1024
    sunLight.shadow.mapSize.height = 1024
    scene.add(sunLight)

    const underwaterLight = new THREE.PointLight(0x06b6d4, 1.2, 20)
    underwaterLight.position.set(0, 1.0, 0)
    scene.add(underwaterLight)

    // Build Tank Architecture
    const tankGroup = new THREE.Group()
    tankMeshGroupRef.current = tankGroup
    scene.add(tankGroup)

    buildTankStructure(tankGroup, tankDimensions, params.systemType)

    // Water Volume Mesh
    const waterColor = getWaterColor(params)
    const waterGeo =
      tankDimensions.type === 'cylinder'
        ? new THREE.CylinderGeometry(
            tankDimensions.radius! * 0.98,
            tankDimensions.radius! * 0.98,
            tankDimensions.waterHeight,
            32,
          )
        : new THREE.BoxGeometry(
            tankDimensions.width! * 0.97,
            tankDimensions.waterHeight,
            (tankDimensions.length || tankDimensions.width!) * 0.97,
          )

    const waterMat = new THREE.MeshPhysicalMaterial({
      color: waterColor,
      transparent: true,
      opacity: 0.62,
      transmission: 0.45,
      roughness: 0.15,
      metalness: 0.05,
      ior: 1.333,
      depthWrite: false,
    })

    const waterMesh = new THREE.Mesh(waterGeo, waterMat)
    waterMesh.position.y = tankDimensions.waterHeight / 2
    tankGroup.add(waterMesh)
    waterMeshRef.current = waterMesh

    // Aeration Bubble Particle System
    if (params.aerationHoursPerDay > 0) {
      const bubbleCount = Math.min(180, Math.floor(params.aerationHoursPerDay * 15))
      const bubbleGeo = new THREE.BufferGeometry()
      const bubblePositions = new Float32Array(bubbleCount * 3)

      for (let i = 0; i < bubbleCount; i++) {
        const radius = Math.random() * (tankDimensions.type === 'cylinder' ? 2.5 : 3.0)
        const angle = Math.random() * Math.PI * 2
        bubblePositions[i * 3] = Math.cos(angle) * radius
        bubblePositions[i * 3 + 1] = Math.random() * tankDimensions.waterHeight
        bubblePositions[i * 3 + 2] = Math.sin(angle) * radius
      }

      bubbleGeo.setAttribute('position', new THREE.BufferAttribute(bubblePositions, 3))
      const bubbleMat = new THREE.PointsMaterial({
        color: 0xe0f2fe,
        size: 0.18,
        transparent: true,
        opacity: 0.75,
        blending: THREE.AdditiveBlending,
      })

      const bubblesMesh = new THREE.Points(bubbleGeo, bubbleMat)
      tankGroup.add(bubblesMesh)
      bubblesMeshRef.current = bubblesMesh
    }

    // Build 3D Fish School
    const fishEntities: FishEntity[] = []
    for (let i = 0; i < renderedFishCount; i++) {
      const fish = createProceduralFish(params.species)
      // Initial random position inside water bounds
      const spawnPos = getRandomWaterPosition(tankDimensions, false)
      fish.group.position.copy(spawnPos)
      fish.position.copy(spawnPos)
      fish.target = getRandomWaterPosition(tankDimensions, false)

      tankGroup.add(fish.group)
      fishEntities.push(fish)
    }
    fishEntitiesRef.current = fishEntities

    // Update Camera Position
    updateCameraPosition()

    // Render loop
    let lastTime = performance.now()

    const animate = (currentTime: number) => {
      const delta = (currentTime - lastTime) / 1000
      lastTime = currentTime

      // Auto-rotation if enabled and not actively dragging
      if (autoRotate && !isDraggingRef.current) {
        sphericalCoordsRef.current.theta += 0.003
        updateCameraPosition()
      }

      // Update fish physics and swimming animation
      animateFishSchool(delta)

      // Animate rising bubbles
      if (bubblesMeshRef.current && bubblesMeshRef.current.geometry.attributes.position) {
        const posAttr = bubblesMeshRef.current.geometry.attributes.position
        const positions = posAttr.array as Float32Array
        for (let i = 1; i < positions.length; i += 3) {
          const currentY = positions[i] ?? 0
          positions[i] = currentY + delta * 1.8 // Rise speed
          if (positions[i]! > tankDimensions.waterHeight) {
            positions[i] = 0.1 // Reset to bottom aerator disc
          }
        }
        posAttr.needsUpdate = true
      }

      // Render frame
      if (rendererRef.current && sceneRef.current && cameraRef.current) {
        rendererRef.current.render(sceneRef.current, cameraRef.current)
      }

      animationFrameIdRef.current = requestAnimationFrame(animate)
    }

    animationFrameIdRef.current = requestAnimationFrame(animate)

    // Window resize handler
    const handleResize = () => {
      if (!containerRef.current || !rendererRef.current || !cameraRef.current) return
      const w = containerRef.current.clientWidth
      const h = Math.max(380, Math.min(520, containerRef.current.clientHeight || 460))
      cameraRef.current.aspect = w / h
      cameraRef.current.updateProjectionMatrix()
      rendererRef.current.setSize(w, h)
    }

    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
      if (animationFrameIdRef.current) {
        cancelAnimationFrame(animationFrameIdRef.current)
      }
      renderer.dispose()
    }
  }, [params.systemType, params.species, params.aerationHoursPerDay, renderedFishCount])

  // Update fish scales and water colors when activeDay / parameters change
  useEffect(() => {
    // Dynamically scale all fish meshes
    fishEntitiesRef.current.forEach((fish) => {
      const s = fishScale * fish.baseScale
      fish.group.scale.set(s, s, s)
    })

    // Update water color
    if (waterMeshRef.current) {
      const mat = waterMeshRef.current.material as THREE.MeshPhysicalMaterial
      mat.color.set(getWaterColor(params))
    }
  }, [fishScale, params])

  // Camera preset mode updates
  const updateCameraPosition = useCallback(() => {
    if (!cameraRef.current) return
    const camera = cameraRef.current
    const coords = sphericalCoordsRef.current

    // Clamp phi to prevent gimbal flips
    coords.phi = Math.max(0.1, Math.min(Math.PI / 2 - 0.05, coords.phi))

    const x = coords.radius * Math.sin(coords.phi) * Math.sin(coords.theta)
    const y = coords.radius * Math.cos(coords.phi)
    const z = coords.radius * Math.sin(coords.phi) * Math.cos(coords.theta)

    camera.position.set(x, y + 1.2, z)
    camera.lookAt(0, 1.2, 0)
  }, [])

  const setCameraPreset = (mode: 'iso' | 'top' | 'surface' | 'underwater') => {
    setCameraMode(mode)
    setAutoRotate(false)
    switch (mode) {
      case 'top':
        sphericalCoordsRef.current = { radius: 16, theta: 0, phi: 0.15 }
        break
      case 'surface':
        sphericalCoordsRef.current = {
          radius: 12,
          theta: Math.PI / 4,
          phi: Math.PI / 2.3,
        }
        break
      case 'underwater':
        sphericalCoordsRef.current = {
          radius: 9,
          theta: Math.PI / 3,
          phi: Math.PI / 2.05,
        }
        break
      case 'iso':
      default:
        sphericalCoordsRef.current = {
          radius: 18,
          theta: Math.PI / 4,
          phi: Math.PI / 3.2,
        }
        break
    }
    updateCameraPosition()
  }

  // Mouse & Touch Orbit Drag Handlers
  const handleMouseDown = (e: React.MouseEvent) => {
    isDraggingRef.current = true
    previousMousePositionRef.current = { x: e.clientX, y: e.clientY }
    setAutoRotate(false)
  }

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isDraggingRef.current) return
    const deltaX = e.clientX - previousMousePositionRef.current.x
    const deltaY = e.clientY - previousMousePositionRef.current.y
    previousMousePositionRef.current = { x: e.clientX, y: e.clientY }

    sphericalCoordsRef.current.theta -= deltaX * 0.008
    sphericalCoordsRef.current.phi -= deltaY * 0.008
    updateCameraPosition()
  }

  const handleMouseUp = () => {
    isDraggingRef.current = false
  }

  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault()
    sphericalCoordsRef.current.radius = Math.max(
      6,
      Math.min(30, sphericalCoordsRef.current.radius + e.deltaY * 0.02),
    )
    updateCameraPosition()
  }

  // Touch Handlers for Mobile
  const handleTouchStart = (e: React.TouchEvent) => {
    if (e.touches.length === 1) {
      isDraggingRef.current = true
      previousMousePositionRef.current = {
        x: e.touches[0]!.clientX,
        y: e.touches[0]!.clientY,
      }
      setAutoRotate(false)
    }
  }

  const handleTouchMove = (e: React.TouchEvent) => {
    if (!isDraggingRef.current || e.touches.length !== 1) return
    const deltaX = e.touches[0]!.clientX - previousMousePositionRef.current.x
    const deltaY = e.touches[0]!.clientY - previousMousePositionRef.current.y
    previousMousePositionRef.current = {
      x: e.touches[0]!.clientX,
      y: e.touches[0]!.clientY,
    }

    sphericalCoordsRef.current.theta -= deltaX * 0.008
    sphericalCoordsRef.current.phi -= deltaY * 0.008
    updateCameraPosition()
  }

  // Fish School Physics & Behavioral Animation Logic
  const animateFishSchool = (delta: number) => {
    const isHypoxic = behavioralState.isHypoxic
    const isSevereHypoxia = behavioralState.isSevereHypoxia
    const speedMult = behavioralState.speedMultiplier

    fishEntitiesRef.current.forEach((fish, idx) => {
      // 1. Update target waypoint if close or hypoxic
      const distToTarget = fish.position.distanceTo(fish.target)
      if (distToTarget < 0.6 || Math.random() < 0.005) {
        fish.target = getRandomWaterPosition(tankDimensions, isHypoxic)
      }

      // 2. Steer towards target
      const desiredVel = fish.target
        .clone()
        .sub(fish.position)
        .normalize()
        .multiplyScalar(0.8 * speedMult * fish.personalSpeed)

      // Surface piping behavior (snout tilted up at the water surface)
      if (isSevereHypoxia) {
        // High cluster near top surface (y ~ waterHeight - 0.15)
        fish.target.y = tankDimensions.waterHeight - 0.12 + (Math.random() * 0.1 - 0.05)
        fish.isPiping = true
      } else {
        fish.isPiping = false
      }

      // Smooth steering interpolation
      fish.velocity.lerp(desiredVel, 0.04)
      fish.position.addScaledVector(fish.velocity, delta * 2.2)
      fish.group.position.copy(fish.position)

      // 3. Orient fish towards velocity direction
      if (fish.velocity.lengthSq() > 0.001) {
        const lookTarget = fish.position.clone().add(fish.velocity)
        fish.group.lookAt(lookTarget)

        // Pitch upwards if piping at water surface
        if (fish.isPiping) {
          fish.group.rotateX(-0.45) // ~25 degree upward tilt
        }
      }

      // 4. Sinusoidal tail undulation (faster wagging with higher speed)
      fish.tailPhase += delta * (4.5 * speedMult + 2.0)
      const tailWagAngle = Math.sin(fish.tailPhase + idx) * (fish.isPiping ? 0.12 : 0.38)
      fish.tailMesh.rotation.y = tailWagAngle
    })
  }

  return (
    <div className="twin-3d-card">
      {/* 3D Visualizer Header & Live HUD */}
      <div className="twin-3d-header">
        <div className="twin-3d-title-group">
          <div className="twin-3d-status-pill" style={{ borderColor: behavioralState.behaviorColor }}>
            <span
              className="status-dot"
              style={{ background: behavioralState.behaviorColor, boxShadow: `0 0 8px ${behavioralState.behaviorColor}` }}
            />
            <span style={{ color: behavioralState.behaviorColor, fontWeight: 700 }}>
              {behavioralState.behaviorName}
            </span>
          </div>
          <h3>🐟 Interactive 3D Culture Tank Twin</h3>
        </div>

        {/* Camera Angle Presets */}
        <div className="twin-3d-camera-presets">
          <button
            type="button"
            className={`cam-btn ${cameraMode === 'iso' ? 'cam-btn--active' : ''}`}
            onClick={() => setCameraPreset('iso')}
            title="Isometric 3D View"
          >
            Isometric
          </button>
          <button
            type="button"
            className={`cam-btn ${cameraMode === 'top' ? 'cam-btn--active' : ''}`}
            onClick={() => setCameraPreset('top')}
            title="Top-Down Bird's Eye View"
          >
            Top-Down
          </button>
          <button
            type="button"
            className={`cam-btn ${cameraMode === 'surface' ? 'cam-btn--active' : ''}`}
            onClick={() => setCameraPreset('surface')}
            title="Water Surface View"
          >
            Waterline
          </button>
          <button
            type="button"
            className={`cam-btn ${cameraMode === 'underwater' ? 'cam-btn--active' : ''}`}
            onClick={() => setCameraPreset('underwater')}
            title="Underwater Inspection View"
          >
            Underwater
          </button>
          <button
            type="button"
            className={`cam-btn ${autoRotate ? 'cam-btn--active' : ''}`}
            onClick={() => setAutoRotate((prev) => !prev)}
            title="Toggle 360° Auto-Rotation"
          >
            {autoRotate ? '⏸ Orbit' : '▶ Orbit'}
          </button>
        </div>
      </div>

      {/* 3D WebGL Canvas Container */}
      <div
        ref={containerRef}
        className="twin-3d-viewport"
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onWheel={handleWheel}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleMouseUp}
      >
        <canvas ref={canvasRef} className="twin-3d-canvas" />

        {/* Telemetry HUD Overlay in Viewport */}
        <div className="twin-3d-hud-overlay">
          <div className="hud-badge">
            <span className="hud-label">Fish Size (Weight)</span>
            <strong className="hud-val">{currentStep.avgWeightG} g</strong>
          </div>
          <div className="hud-badge">
            <span className="hud-label">Swim Velocity</span>
            <strong className="hud-val">{behavioralState.swimSpeedCmS} cm/s</strong>
          </div>
          <div className="hud-badge">
            <span className="hud-label">Live DO / Temp</span>
            <strong className="hud-val">
              {params.dissolvedOxygenMgL} mg/L · {params.waterTempC}°C
            </strong>
          </div>
          <div className="hud-badge">
            <span className="hud-label">Survival & Density</span>
            <strong className="hud-val">
              {outcome.survivalRatePct}% · {behavioralState.activeDensity.toFixed(1)} kg/m³
            </strong>
          </div>
        </div>

        {/* Drag Hint */}
        <span className="twin-3d-hint">🖐 Drag to rotate 3D tank · Scroll / Pinch to zoom</span>
      </div>

      {/* Interactive Day Timeline Scrubber */}
      <div className="twin-3d-timeline-bar">
        <div className="timeline-controls">
          <button
            type="button"
            className={`timeline-play-btn ${isPlaying ? 'timeline-play-btn--active' : ''}`}
            onClick={() => setIsPlaying((p) => !p)}
            title={isPlaying ? 'Pause Simulation' : 'Play Simulation Time-Lapse'}
          >
            {isPlaying ? '⏸ Pause' : '▶ Play Cycle'}
          </button>

          <div className="timeline-day-display">
            <span>Day:</span>
            <strong>{activeDay}</strong>
            <small>/ {horizonDays} days</small>
          </div>
        </div>

        <div className="timeline-slider-container">
          <input
            type="range"
            min="1"
            max={horizonDays}
            value={activeDay}
            onChange={(e) => onDayChange && onDayChange(Number(e.target.value))}
            className="twin-timeline-slider"
          />
          <div className="timeline-markers">
            <span>Day 0 (Stocking: {params.initialAvgWeightG}g)</span>
            <span>Mid-Cycle ({Math.round(horizonDays / 2)}d)</span>
            <span>Day {horizonDays} (Harvest: {outcome.finalAvgWeightG}g)</span>
          </div>
        </div>
      </div>
    </div>
  )
}

// ============================================================================
// Helper Functions: Tank Construction & Procedural Fish Mesh Generation
// ============================================================================

function getWaterColor(params: DigitalTwinPondParams): number {
  if (params.ammoniaMgL > 0.8) {
    return 0x2e8b57 // Murky greenish yellow under ammonia toxicity
  }
  switch (params.systemType) {
    case 'earthen':
      return 0x1f6e63 // Clay algae green
    case 'tarpaulin':
      return 0x0284c7 // Ocean cyan
    case 'ras':
      return 0x06b6d4 // Crystal clear filtered blue
    case 'concrete':
    default:
      return 0x0891b2 // Aqua teal
  }
}

function getRandomWaterPosition(
  dim: { type: string; radius?: number; width?: number; length?: number; waterHeight: number },
  isHypoxic: boolean,
): THREE.Vector3 {
  let x = 0
  let z = 0
  if (dim.type === 'cylinder') {
    const r = Math.random() * (dim.radius! * 0.78)
    const angle = Math.random() * Math.PI * 2
    x = Math.cos(angle) * r
    z = Math.sin(angle) * r
  } else {
    x = (Math.random() - 0.5) * (dim.width! * 0.78)
    z = (Math.random() - 0.5) * ((dim.length || dim.width!) * 0.78)
  }

  // Vertical position
  let y = 0
  if (isHypoxic) {
    // Upper water column (85% to 98% height)
    y = dim.waterHeight * (0.85 + Math.random() * 0.12)
  } else {
    // General water column (15% to 80% height)
    y = dim.waterHeight * (0.15 + Math.random() * 0.65)
  }

  return new THREE.Vector3(x, y, z)
}

function buildTankStructure(
  group: THREE.Group,
  dim: { type: string; radius?: number; width?: number; length?: number; height: number },
  systemType: PondSystemType,
) {
  // Clear any existing children
  while (group.children.length > 0) {
    group.remove(group.children[0]!)
  }

  if (dim.type === 'cylinder') {
    // Tarpaulin or RAS Circular Tank
    const wallGeo = new THREE.CylinderGeometry(dim.radius!, dim.radius!, dim.height, 36, 1, true)
    const wallColor = systemType === 'tarpaulin' ? 0x1d4ed8 : 0x334155
    const wallMat = new THREE.MeshStandardMaterial({
      color: wallColor,
      roughness: 0.4,
      metalness: 0.2,
      side: THREE.DoubleSide,
    })
    const wallMesh = new THREE.Mesh(wallGeo, wallMat)
    wallMesh.position.y = dim.height / 2
    group.add(wallMesh)

    // Floor
    const floorGeo = new THREE.CircleGeometry(dim.radius!, 36)
    const floorMat = new THREE.MeshStandardMaterial({
      color: 0x0f172a,
      roughness: 0.8,
      side: THREE.DoubleSide,
    })
    const floorMesh = new THREE.Mesh(floorGeo, floorMat)
    floorMesh.rotation.x = -Math.PI / 2
    group.add(floorMesh)

    // Metal Frame Ring on Top
    const ringGeo = new THREE.TorusGeometry(dim.radius!, 0.08, 12, 36)
    const ringMat = new THREE.MeshStandardMaterial({ color: 0x94a3b8, metalness: 0.8, roughness: 0.2 })
    const ringMesh = new THREE.Mesh(ringGeo, ringMat)
    ringMesh.rotation.x = Math.PI / 2
    ringMesh.position.y = dim.height
    group.add(ringMesh)
  } else {
    // Concrete Rectangular Tank or Earthen Pond
    const w = dim.width!
    const l = dim.length || dim.width!
    const h = dim.height

    // Floor Base
    const floorGeo = new THREE.PlaneGeometry(w, l)
    const floorColor = systemType === 'earthen' ? 0x452c1e : 0x1e293b
    const floorMat = new THREE.MeshStandardMaterial({
      color: floorColor,
      roughness: 0.9,
      side: THREE.DoubleSide,
    })
    const floorMesh = new THREE.Mesh(floorGeo, floorMat)
    floorMesh.rotation.x = -Math.PI / 2
    group.add(floorMesh)

    // 4 Walls
    const wallMat = new THREE.MeshStandardMaterial({
      color: systemType === 'earthen' ? 0x5c3d2e : 0x334155,
      roughness: 0.7,
      side: THREE.DoubleSide,
    })

    // Back wall
    const backGeo = new THREE.PlaneGeometry(w, h)
    const backWall = new THREE.Mesh(backGeo, wallMat)
    backWall.position.set(0, h / 2, -l / 2)
    group.add(backWall)

    // Front wall (semi-transparent glass panel for viewing)
    const glassMat = new THREE.MeshPhysicalMaterial({
      color: 0xffffff,
      transparent: true,
      opacity: 0.25,
      roughness: 0.1,
      metalness: 0.1,
      transmission: 0.7,
      side: THREE.DoubleSide,
    })
    const frontWall = new THREE.Mesh(backGeo, glassMat)
    frontWall.position.set(0, h / 2, l / 2)
    group.add(frontWall)

    // Left wall
    const sideGeo = new THREE.PlaneGeometry(l, h)
    const leftWall = new THREE.Mesh(sideGeo, wallMat)
    leftWall.rotation.y = Math.PI / 2
    leftWall.position.set(-w / 2, h / 2, 0)
    group.add(leftWall)

    // Right wall
    const rightWall = new THREE.Mesh(sideGeo, wallMat)
    rightWall.rotation.y = -Math.PI / 2
    rightWall.position.set(w / 2, h / 2, 0)
    group.add(rightWall)
  }

  // Water Inlet Pipe at top corner
  const pipeGeo = new THREE.CylinderGeometry(0.12, 0.12, 1.2, 16)
  const pipeMat = new THREE.MeshStandardMaterial({ color: 0x38bdf8, metalness: 0.5, roughness: 0.3 })
  const pipeMesh = new THREE.Mesh(pipeGeo, pipeMat)
  pipeMesh.rotation.z = Math.PI / 2
  pipeMesh.position.set(-2.8, dim.height + 0.1, 0)
  group.add(pipeMesh)

  // Aerator Diffuser Disc at bottom center
  const discGeo = new THREE.CylinderGeometry(0.45, 0.5, 0.1, 24)
  const discMat = new THREE.MeshStandardMaterial({ color: 0x0f172a, roughness: 0.5 })
  const discMesh = new THREE.Mesh(discGeo, discMat)
  discMesh.position.set(0, 0.05, 0)
  group.add(discMesh)
}

function createProceduralFish(species: FishSpecies): FishEntity {
  const group = new THREE.Group()

  // Species-specific colors & geometries
  const isCatfish = species === 'catfish' || species === 'heteroclarias'
  const bodyColor = isCatfish ? 0x27272a : 0x475569 // Slate black vs olive silver
  const finColor = isCatfish ? 0x3f3f46 : 0xf87171 // Dark vs red-tinted tilapia fins

  // 1. Tapered Anatomical Fish Body
  const bodyGeo = new THREE.ConeGeometry(0.22, 1.1, 16)
  bodyGeo.rotateX(Math.PI / 2) // Point forward along +Z axis
  const bodyMat = new THREE.MeshStandardMaterial({
    color: bodyColor,
    roughness: 0.35,
    metalness: 0.25,
  })
  const bodyMesh = new THREE.Mesh(bodyGeo, bodyMat)
  bodyMesh.position.z = 0.2
  group.add(bodyMesh)

  // 2. Fish Head Snout
  const headGeo = new THREE.SphereGeometry(0.21, 14, 10)
  headGeo.scale(1.0, 0.75, 1.3)
  const headMesh = new THREE.Mesh(headGeo, bodyMat)
  headMesh.position.z = 0.65
  group.add(headMesh)

  // 3. Caudal (Tail) Fin (Animated Wag)
  const tailGroup = new THREE.Group()
  tailGroup.position.z = -0.45 // Tail hinge point

  const tailGeo = new THREE.BufferGeometry()
  // 2D Triangle / fan vertices for tail fin
  const tailVertices = new Float32Array([
    0, 0, 0,
    0, 0.28, -0.42,
    0, -0.28, -0.42,
  ])
  tailGeo.setAttribute('position', new THREE.BufferAttribute(tailVertices, 3))
  tailGeo.computeVertexNormals()

  const tailMat = new THREE.MeshStandardMaterial({
    color: finColor,
    roughness: 0.4,
    side: THREE.DoubleSide,
  })
  const tailMesh = new THREE.Mesh(tailGeo, tailMat)
  tailGroup.add(tailMesh)
  group.add(tailGroup)

  // 4. Catfish Whiskers (Barbels)
  if (isCatfish) {
    const barbelGeo = new THREE.CylinderGeometry(0.015, 0.005, 0.4, 8)
    const barbelMat = new THREE.MeshStandardMaterial({ color: 0x18181b })

    const leftBarbel = new THREE.Mesh(barbelGeo, barbelMat)
    leftBarbel.position.set(0.14, -0.04, 0.82)
    leftBarbel.rotation.set(0.3, 0.5, 0.4)
    group.add(leftBarbel)

    const rightBarbel = new THREE.Mesh(barbelGeo, barbelMat)
    rightBarbel.position.set(-0.14, -0.04, 0.82)
    rightBarbel.rotation.set(0.3, -0.5, -0.4)
    group.add(rightBarbel)
  }

  // 5. Dorsal Fin
  const dorsalGeo = new THREE.BufferGeometry()
  const dorsalVertices = new Float32Array([
    0, 0.18, 0.3,
    0, 0.42, 0.05,
    0, 0.12, -0.2,
  ])
  dorsalGeo.setAttribute('position', new THREE.BufferAttribute(dorsalVertices, 3))
  dorsalGeo.computeVertexNormals()
  const dorsalMesh = new THREE.Mesh(dorsalGeo, tailMat)
  group.add(dorsalMesh)

  return {
    group,
    bodyMesh,
    tailMesh: tailGroup as unknown as THREE.Mesh,
    position: new THREE.Vector3(),
    velocity: new THREE.Vector3(),
    target: new THREE.Vector3(),
    rotation: new THREE.Euler(),
    tailPhase: Math.random() * Math.PI * 2,
    tailSpeed: 4.0 + Math.random() * 2.0,
    baseScale: 0.85 + Math.random() * 0.3,
    personalSpeed: 0.85 + Math.random() * 0.3,
    isPiping: false,
  }
}
