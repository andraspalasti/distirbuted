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

- **c)**: Now there is no guarantee that each node will actually get the 
  messages between each other. So we need to keep track of which message was
  actually received by which node.
  Each node does the same:
    1. When a broadcast is received, it is added to the list of recieved messages.
      If it didn't came from another node, then it is to be distributed to all.
    1. The messages that are to be distributed are only marked as sent after 
      the destination node confirms that it recieved it.

- **d)**: The previous solution is working fine in this case too, the only
  change is that the distribution of messages are done less frequently.

- **e)**: In this case to reduce the number of messages between nodes, we batch
  them together.

