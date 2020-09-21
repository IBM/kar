This example uses KAR's Actor Programming Model to implement
Dijkstra's solution to the Dining Philosophers problem
(https://en.wikipedia.org/wiki/Dining_philosophers_problem).

The Philosophers and their Forks are all actors and interact via actor
method invocations to implement the distributed protocol that ensures
no Philosopher starves.

Philosophers use actor reminders (time triggered callbacks) to trigger
their actions.

Fault tolerance is provided by checkpointing actor state and the
at-least-once invocation semantics provided by KAR.

A Cafe may contain an arbitrary number of tables of Philosophers. Each
Cafe tracks its occupancy and generates messages when it seats new
tables or when a sated Philosopher leaves.

To run the example locally, first do a `mvn package`.
Then in one window start up the server code:
```shell
kar run -app dpj -actors Cafe,Fork,Philosopher mvn liberty:run
```
In a second window, use the `kar` cli to invite some Philosopers to diner:
```shell
# Invite 10 Philosophers to a feast of 20 servings each
kar invoke -app dpj Cafe "Cafe de Flore" seatTable 10 20
```
