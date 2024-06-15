The [fly.io distributed system challanges](https://fly.io/dist-sys/) solved in go.

# Explanations

## Unique ID Generation

For generating a globally unique ID, we use the name of each node which is given
on initalization, also each node keeps track of a counter value which is 
incremented, on each generate request.

We concatenate these two, and we get the globally unique ID that we send back. 


## Broadcast challenges

- **b)**: Because we know that there will be no network partitions we can
 assume that all messages will be delivered. When a node recieves a broadcast
 it checks if it came from another node or not, if another node sent it, then
 it doesn't need to broadcast it further, else it needs to sent to all others.
