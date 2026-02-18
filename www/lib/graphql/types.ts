export interface AvailableBackend {
  type: string;
  supported: boolean;
  registered: boolean;
}

export interface Backend {
  id: string;
  name: string;
  description: string;
  url: string;
  enabled: boolean;
  supported: boolean;
  servers: Server[];
  peers: Peer[];
  foreignServers: ForeignServer[];
  createUser: User | null;
  updateUser: User | null;
  deleteUser: User | null;
  createdAt: string;
  updatedAt: string;
  deletedAt: string | null;
}

export interface User {
  id: string;
  email: string;
  createdAt: string;
  updatedAt: string;
  peers?: Peer[];
  servers?: Server[];
}

export interface ServerHook {
  command: string;
  runOnPreUp: boolean;
  runOnPostUp: boolean;
  runOnPreDown: boolean;
  runOnPostDown: boolean;
}

export interface ServerInterfaceStats {
  rxBytes: number;
  txBytes: number;
}

export interface Server {
  id: string;
  name: string;
  description: string;
  address: string;
  listenPort: number | null;
  dns: string[] | null;
  mtu: number;
  firewallMark: number | null;
  enabled: boolean;
  running: boolean;
  publicKey: string;
  hooks: ServerHook[] | null;
  interfaceStats: ServerInterfaceStats | null;
  peers: Peer[] | null;
  backend: Backend;
  createUser: User | null;
  updateUser: User | null;
  deleteUser: User | null;
  createdAt: string;
  updatedAt: string;
  deletedAt: string | null;
}

export interface PeerHook {
  command: string;
  runOnCreate: boolean;
  runOnDelete: boolean;
  runOnUpdate: boolean;
}

export interface PeerStats {
  endpoint: string | null;
  lastHandshakeTime: string | null;
  protocolVersion: number;
  receiveBytes: number;
  transmitBytes: number;
}

export interface Peer {
  id: string;
  name: string;
  description: string;
  publicKey: string;
  presharedKey: string;
  endpoint: string;
  allowedIPs: string[] | null;
  persistentKeepalive: number | null;
  hooks: PeerHook[] | null;
  stats: PeerStats | null;
  server: Server;
  backend: Backend;
  createUser: User | null;
  updateUser: User | null;
  deleteUser: User | null;
  createdAt: string;
  updatedAt: string;
  deletedAt: string | null;
}

export interface ForeignInterface {
  name: string;
  addresses: string[];
  mtu: number;
}

export interface ForeignPeer {
  publicKey: string;
  endpoint: string | null;
  allowedIps: string[] | null;
  lastHandshakeTime: string | null;
  persistentKeepAliveInterval: number;
  protocolVersion: number;
  receiveBytes: number;
  transmitBytes: number;
}

export interface ForeignServer {
  name: string;
  type: string;
  publicKey: string;
  listenPort: number;
  firewallMark: number;
  foreignInterface: ForeignInterface;
  peers: ForeignPeer[];
  backend: Backend;
}

export interface SignInPayload {
  token: string;
  expiresAt: string;
  expiresIn: number;
}

export interface GenerateWireguardKeyPayload {
  privateKey: string;
  publicKey: string;
}

// Input types
export interface ServerHookInput {
  command: string;
  runOnPreUp: boolean;
  runOnPostUp: boolean;
  runOnPreDown: boolean;
  runOnPostDown: boolean;
}

export interface PeerHookInput {
  command: string;
  runOnCreate: boolean;
  runOnDelete: boolean;
  runOnUpdate: boolean;
}

export interface CreateServerInput {
  name: string;
  address: string;
  backendId: string;
  description?: string;
  dns?: string[];
  enabled?: boolean;
  firewallMark?: number;
  hooks?: ServerHookInput[];
  listenPort?: number;
  mtu?: number;
  privateKey?: string;
}

export interface UpdateServerInput {
  id: string;
  name?: string;
  address?: string;
  description?: string;
  dns?: string[];
  enabled?: boolean;
  firewallMark?: number;
  hooks?: ServerHookInput[];
  listenPort?: number;
  mtu?: number;
  privateKey?: string;
}

export interface CreatePeerInput {
  serverId: string;
  name: string;
  publicKey: string;
  allowedIPs: string[];
  description?: string;
  endpoint?: string;
  hooks?: PeerHookInput[];
  persistentKeepalive?: number;
  presharedKey?: string;
}

export interface UpdatePeerInput {
  id: string;
  name?: string;
  publicKey?: string;
  allowedIPs?: string[];
  description?: string;
  endpoint?: string;
  hooks?: PeerHookInput[];
  persistentKeepalive?: number;
  presharedKey?: string;
}

export interface CreateUserInput {
  email: string;
  password: string;
}

export interface UpdateUserInput {
  id: string;
  email?: string;
  password?: string;
}
