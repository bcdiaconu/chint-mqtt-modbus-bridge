# Per-Group Polling Configuration

## Overview

Starting with version 2.2, the MQTT-Modbus Bridge supports **per-group polling intervals**. This allows each register group to be polled at its own optimal frequency, improving efficiency and reducing unnecessary Modbus traffic.

## Why Per-Group Polling?

Different types of measurements have different update requirements:

- **Instant Measurements** (voltage, current, power) change frequently → Poll every 1 second
- **Energy Counters** (kWh) change slowly → Poll every 5-10 seconds
- **Status Registers** (device state, errors) rarely change → Poll every 30-60 seconds

With device-level polling, all groups were polled at the same interval (typically 1 second), generating unnecessary Modbus traffic and gateway load.

## Configuration

### Before (Version 2.1 and earlier)

```yaml
devices:
  energy_meter_mains:
    rtu:
      slave_id: 11
      poll_interval: 1000  # ❌ All groups polled at same interval
    
    modbus:
      register_groups:
        instant:
          name: "Instant Measurements"
          enabled: true
          # ... registers ...
        
        energy:
          name: "Energy Counters"
          enabled: true
          # ... registers ...
```

**Problem**: Both instant and energy groups polled every 1 second, even though energy counters don't need frequent updates.

### After (Version 2.2+)

```yaml
devices:
  energy_meter_mains:
    rtu:
      slave_id: 11  # ✅ No device-level poll_interval
    
    modbus:
      register_groups:
        instant:
          name: "Instant Measurements"
          enabled: true
          poll_interval: 1000  # ✅ Poll instant values every 1 second
          # ... registers ...
        
        energy:
          name: "Energy Counters"
          enabled: true
          poll_interval: 5000  # ✅ Poll energy counters every 5 seconds
          # ... registers ...
```

**Benefit**:

- Instant measurements: 1 poll/second = 3,600 polls/hour
- Energy counters: 1 poll/5 seconds = 720 polls/hour
- **Total reduction**: From 7,200 polls/hour to 4,320 polls/hour (40% reduction)

## Scheduler Architecture

### How It Works

The bridge uses a **GroupScheduler** that:

1. Tracks each group's `poll_interval` independently
2. Maintains last execution time for each group
3. Checks every 100ms which groups are due for execution
4. Executes groups **sequentially** (never in parallel) to prevent race conditions

### Sequential Execution Guarantee

Even though groups have different intervals, **execution is always sequential**:

```md
Time    Group            Action
─────── ─────────────── ─────────────────────────
0ms     instant_1       Execute (150ms)
150ms   instant_2       Execute (140ms)
290ms   -               Wait
1000ms  instant_1       Execute (150ms)  ← Due again
1150ms  instant_2       Execute (140ms)  ← Due again
1290ms  -               Wait
5000ms  energy_1        Execute (120ms)  ← Due for first time
5120ms  instant_1       Execute (150ms)  ← Also due
```

**Key Points**:

- ✅ Groups execute **one at a time** (mutex-protected)
- ✅ No response collision (validated SlaveID/FunctionCode)
- ✅ Stale responses cleared before each request
- ✅ Execution time tracked for diagnostics

See [SEQUENTIAL_EXECUTION.md](SEQUENTIAL_EXECUTION.md) for detailed timing analysis.

## Configuration Rules

### Required

Each `register_group` **must** have a `poll_interval`:

```yaml
register_groups:
  instant:
    poll_interval: 1000  # ✅ Required (milliseconds)
```

### Validation

The configuration validator checks:

- ✅ `poll_interval` is **present** (not optional)
- ✅ `poll_interval` > 0 (must be positive)
- ✅ `poll_interval` ≤ 300,000 ms (max 5 minutes)

**Error example**:

```log
❌ ERROR: poll_interval must be positive for register group 'instant' (got 0 ms)
❌ ERROR: poll_interval too large for register group 'energy' (got 600000 ms, max 300000 ms)
```

## Recommended Poll Intervals

| Register Type          | Recommended Interval | Reason                                    |
|------------------------|---------------------|-------------------------------------------|
| Voltage, Current       | 1000 ms (1 second)  | Real-time monitoring                      |
| Active Power           | 1000 ms (1 second)  | Real-time consumption tracking            |
| Power Factor           | 2000 ms (2 seconds) | Changes slowly                            |
| Frequency              | 5000 ms (5 seconds) | Stable (rarely changes)                   |
| Energy Counters (kWh)  | 5000-10000 ms       | Accumulates slowly                        |
| Status Registers       | 30000 ms (30 sec)   | Device state rarely changes               |
| Temperature Sensors    | 60000 ms (1 min)    | Changes very slowly                       |

## Migration Guide

### Step 1: Update Configuration

**Before**:

```yaml
rtu:
  slave_id: 11
  poll_interval: 1000  # ← Remove this
```

**After**:

```yaml
rtu:
  slave_id: 11  # ← Keep only slave_id
```

### Step 2: Add poll_interval to Each Group

**Before**:

```yaml
register_groups:
  instant:
    enabled: true
```

**After**:

```yaml
register_groups:
  instant:
    enabled: true
    poll_interval: 1000  # ← Add this (required)
```

### Step 3: Optimize Intervals

Review each group and set appropriate intervals:

```yaml
register_groups:
  instant:
    poll_interval: 1000  # Fast updates for real-time data
  
  energy:
    poll_interval: 5000  # Slower updates for counters
  
  status:
    poll_interval: 30000 # Very slow updates for status
```

### Step 4: Test Configuration

```bash
# Validate configuration
./mqtt-modbus-bridge --config config.yaml --validate

# Check logs for scheduling info
# Expected output:
# 📅 Scheduled group 'energy_meter_mains_instant' with interval: 1s (1000 ms)
# 📅 Scheduled group 'energy_meter_mains_energy' with interval: 5s (5000 ms)
# 📅 Group scheduler initialized with 2 groups (check interval: 100ms)
```

## Performance Benefits

### Example: 2 Devices, 2 Groups Each

**Before (device-level polling)**:

- 4 groups × 1 poll/second = 4 polls/second
- 4 × 3,600 = **14,400 polls/hour**

**After (group-level polling)**:

- 2 instant groups × 1 poll/second = 2 polls/second
- 2 energy groups × 1 poll/5 seconds = 0.4 polls/second
- Total: 2.4 polls/second = **8,640 polls/hour**

**Result**: **40% reduction** in Modbus traffic

### Additional Benefits

- ✅ **Reduced Gateway Load**: Fewer MQTT messages published
- ✅ **Lower Bus Utilization**: Less RS-485 traffic
- ✅ **Better Battery Life**: Reduced power consumption (for battery-powered gateways)
- ✅ **Improved Reliability**: Less chance of bus collisions
- ✅ **Faster Response**: Critical groups polled more frequently

## Debugging

### Enable Debug Logging

```yaml
logging:
  level: "trace"
```

### Expected Log Messages

**Scheduler Initialization**:

```log
📅 Scheduled group 'energy_meter_mains_instant' with interval: 1s (1000 ms)
📅 Scheduled group 'energy_meter_mains_energy' with interval: 5s (5000 ms)
📅 Group scheduler initialized with 2 groups (check interval: 100ms)
🔄 Group scheduler started (check interval: 100ms)
```

**Group Execution**:

```log
⏰ Groups due for execution: [energy_meter_mains_instant]
🔄 Executing group 'energy_meter_mains_instant'...
🔄 Executing group 'energy_meter_mains_instant' (Slave 11, Addr 0x2000, Count 34)
✅ Group 'energy_meter_mains_instant' (Slave 11) read successful (68 bytes)
✅ Group 'energy_meter_mains_instant' executed successfully in 145ms (6 registers)
```

**Timing Analysis**:

```log
⏰ Groups due for execution: [energy_meter_mains_instant, energy_meter_mains_energy]
🔄 Executing group 'energy_meter_mains_instant'...
✅ Group 'energy_meter_mains_instant' executed successfully in 145ms
🔄 Executing group 'energy_meter_mains_energy'...
✅ Group 'energy_meter_mains_energy' executed successfully in 120ms
```

## Thread Safety & Sequential Execution

### Execution Guarantee

The GroupScheduler ensures **only one group executes at a time**, even if multiple groups become due simultaneously:

```go
// CRITICAL: executionMutex ensures sequential execution
s.executionMutex.Lock()
defer s.executionMutex.Unlock()
```

**Why this matters**:

- ✅ Prevents concurrent Modbus requests (serial communication is sequential)
- ✅ Avoids race conditions in response handling
- ✅ Prevents response mix-ups between devices/groups
- ✅ Guarantees stable circuit breaker behavior

### Execution Flow

```md
Tick 1 (T=0ms):
  ├─ Group A due? → YES → Lock → Execute → Unlock (150ms)
  └─ Group B due? → YES → Wait for lock...

Tick 2 (T=100ms):
  └─ Group B continues → Lock → Execute → Unlock (120ms)

Tick 3 (T=200ms):
  └─ (all groups completed, scheduler idle)
```

**Log Evidence**:

```log
⏰ Groups due for execution: [instant_group, energy_group]
🔄 Executing group 'instant_group'...          ← Lock acquired
✅ Group 'instant_group' executed in 145ms     ← Lock released
🔄 Executing group 'energy_group'...           ← Lock acquired (next group waits)
✅ Group 'energy_group' executed in 120ms      ← Lock released
```

### Race Condition Prevention

**Problem Without Mutex** (old implementation):

```md
T=0ms:  Scheduler checks → Group A due, Group B due
T=1ms:  Group A starts → Sends Modbus request to Slave 11
T=2ms:  Group B starts → Sends Modbus request to Slave 1 (concurrent!)
T=50ms: Response arrives from Slave 11 → BUT Group B is expecting Slave 1!
        ⚠️ ERROR: "Received unexpected response (Slave=11) but expecting (Slave=1)"
        ❌ Both groups timeout → Circuit breaker opens
```

**Solution With Mutex** (current implementation):

```md
T=0ms:   Scheduler checks → Group A due, Group B due
T=1ms:   Group A acquires lock → Sends Modbus request to Slave 11
T=2ms:   Group B tries lock → BLOCKED (waits)
T=50ms:  Response arrives from Slave 11 → Correctly routed to Group A ✅
T=150ms: Group A releases lock
T=151ms: Group B acquires lock → Sends Modbus request to Slave 1
T=200ms: Response arrives from Slave 1 → Correctly routed to Group B ✅
```

**Result**: Zero race conditions, 100% response accuracy

### Common Issues

#### Error: "poll_interval is required"

**Symptom**:

```log
❌ ERROR: poll_interval must be positive for register group 'instant' (got 0 ms)
```

**Solution**: Add `poll_interval` to the group:

```yaml
register_groups:
  instant:
    poll_interval: 1000  # ← Add this
```

#### Warning: "Groups not executing"

**Symptom**: No log messages like `⏰ Groups due for execution`

**Causes**:

1. Group not enabled: `enabled: false`
2. Interval too large: `poll_interval: 999999999`
3. Configuration not loaded

**Solution**: Check configuration and restart bridge.

## API Changes

### StrategyExecutor

**New Method**:

```go
// ExecuteGroup executes a single register group by key
ExecuteGroup(ctx context.Context, groupKey string) (map[string]*CommandResult, error)

// GetGroupIntervals returns poll intervals for all groups
GetGroupIntervals() map[string]int
```

### GroupScheduler (New Package)

```go
// NewGroupScheduler creates a scheduler with per-group intervals
scheduler := scheduler.NewGroupScheduler(executor, groupIntervals)

// Start with callback for publishing results
scheduler.Start(ctx, func(ctx context.Context, results map[string]*CommandResult) {
    // Publish results to Home Assistant
})
```

## Backward Compatibility

⚠️ **Breaking Change**: Configurations from version 2.1 and earlier **will not work** without modification.

**Required Migration**: Move `poll_interval` from `rtu` to each `register_group`.

**Migration Script**: Use `scripts/migrate-config.sh` to automatically update your configuration:

```bash
./scripts/migrate-config.sh config.yaml > config-v2.2.yaml
```

## See Also

- [Configuration Documentation](CONFIG.md)
- [Sequential Execution Documentation](SEQUENTIAL_EXECUTION.md)
- [Architecture Documentation](ARCHITECTURE.md)
- [DDSU666-H Register Map](DDSU666-H.md)

## Version History

- **v2.2.0** - Introduced per-group polling
- **v2.1.0** - Device-based configuration with register groups
- **v2.0.0** - Multi-device support
- **v1.x** - Original single-device implementation
