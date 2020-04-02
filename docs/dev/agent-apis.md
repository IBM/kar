## Design of KAR Agents (Virtual Actors)

## Reminder Services

Reminders provide a facility for a currently running agent to schedule
future invocations of itself. Once the runtime has accepted a
reminder request, it is guarenteed to be delivered to an instance of
the agent at some point in time after the requested deadline.

Reminders can carry an optional data payload.

Reminders can be either one-shot or periodic.

Periodic reminders can optionally be configured to only be delivered
while the target agent is already active (in memory). When an agent is
passivated, these reminders are suspended and automatically
re-initiated once the agent is activated (as if they had just been
registered with the runtime by the agent).

Periodic reminders must be explictly cancelled by the agent to prevent
them from firing.

One shot reminders are automatically cancelled by the runtime after
they are delivered to the target agent.

All reminders are scheduled using absolute time.
Reminders will be delivered no sooner than the requested time.

The target deadline for the next invocation of a periodic timer is
determined by the runtime by adding the period to the current time
when the reminder was scheudled for delivery to the agent (last fire time).

Strawman API
```
schedulePeriodicReminder(id, entryPoint, period, data, shouldActivatePassiveAgent)

scheduleOneShotReminder(id, entryPoint, deadline, data)

cancelReminder(id)

getReminders() => collection[id]

getReminder(id) => object describing reminder parameters

```



