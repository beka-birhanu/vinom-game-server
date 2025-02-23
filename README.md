# Vinom Game Session Manager and Game Server

We have two main components here: the **Session Manager** and the **Game Server**. The **Session Manager** is responsible for sending game server responses to clients and forwarding client requests to the respective game servers.

## How does it work?

When a game match is found, the matching server sends a new gamer request to the **Session Manager**. The **Session Manager** creates a new instance of the game and starts it, while also adding the players to a session list. The **Session Manager** has access to a UDP socket manager.

When a client tries to connect to the socket, the **Socket Manager** asks the **Session Manager** to authenticate the client. If there is a game session for that client, the **Session Manager** allows the connection, and the **Socket Manager** establishes a secure connection.

When the client sends a request, the **Socket Manager** passes it to the **Session Manager**. The **Session Manager** maps the client to the respective game server and forwards the message. The **Game Server** processes the request and provides the response to the **Session Manager**, which then sends the response to the appropriate clients.

### Unreliable Network Communication

There’s one issue left without discussion: we are communicating over an unreliable network (UDP). Messages might be lost or retransmitted, and we need to handle this properly to avoid negative effects on the game. The key things communicated between the client and server are the game state and move action requests.

For the client side, the solution is relatively simple. We attach a version number to the game state. If we receive a game state with the same or lower version, we ignore it.

However, for the server logic, we can’t just use a version number. Why? Imagine two clients sending a move action at the same time, both sending the same version number. One of them gets executed first, causing the game version to change. This means the second request might be rejected, but it doesn’t have to be. 

To solve this, I came up with the idea of giving each client their own version number. Instead of using an additional version variable, we can use the client's current position as the version. When a client sends an action request, they will include their current position on the grid as their own version number under their ownership. This ensures that no one else can change it, preventing the two simultaneous requests from blocking each other.

## What is left?

We need to introduce **Kubernetes** to manage the spawning of game servers.

---

Note: Dancing is key.

![gopher](./assets/logo/gopher-dance-long-3x.gif)

