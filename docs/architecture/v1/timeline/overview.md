# Overview

The foreground process monitor currently suffers from a major drawback: it is always one step behind because it only reports once we have switched to another application. We want to be able to display the current foreground application in addition to applications which were used previously

# Design goals

- Able to query the database directly and get a nice sequence of foreground applications. We currently do the normalization client side

- Able to see the current foreground application

- Able to reconstruct the timeline in the case of data loss

# Approach

Basically a running projection (?) of our foreground process event store.

1. Poll every *x* seconds for the foreground application and store it

2. Combine consecutive foreground applications with the same PID. Within a unit of time, if there are two foreground applications, then we pick the last one. For example, if the user quickly switches from VSCode to Codex, the current foreground application is Codex and we drop the VSCode entry

3. Reconcile the timeline that is persisted with the timeline that is computed. Concretely, after we have polled for the foreground process:
   - Does the latest entry in the timeline have the same PID? Add to it
   - Does the latest entry in the timeline have a different PID?
      - Is it >= *y* units before (configurable granularity)? Set end time of previous entry and create new entry
      - Is it < *y* units before (configurable granularity)? Delete it and create a new entry

4. Create an embedding for the entry. Embeddings should be linked to the timeline entry and it should cascade delete
