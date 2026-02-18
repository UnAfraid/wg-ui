import { gql } from "@apollo/client";

export const VIEWER_QUERY = gql`
  query Viewer {
    viewer {
      id
      email
      createdAt
      updatedAt
    }
  }
`;

export const SERVERS_QUERY = gql`
  query Servers($query: String, $enabled: Boolean) {
    servers(query: $query, enabled: $enabled) {
      id
      name
      description
      address
      listenPort
      dns
      mtu
      enabled
      running
      publicKey
      createdAt
      updatedAt
      backend {
        id
        name
      }
      peers {
        id
      }
      interfaceStats {
        rxBytes
        txBytes
      }
    }
  }
`;

export const SERVER_QUERY = gql`
  query Server($id: ID!) {
    node(id: $id) {
      ... on Server {
        id
        name
        description
        address
        listenPort
        dns
        mtu
        firewallMark
        enabled
        running
        publicKey
        createdAt
        updatedAt
        backend {
          id
          name
        }
        hooks {
          command
          runOnCreate
          runOnDelete
          runOnStart
          runOnStop
          runOnUpdate
        }
        interfaceStats {
          rxBytes
          txBytes
        }
        peers {
          id
          name
          description
          publicKey
          endpoint
          allowedIPs
          persistentKeepalive
          presharedKey
          createdAt
          updatedAt
          stats {
            endpoint
            lastHandshakeTime
            receiveBytes
            transmitBytes
            protocolVersion
          }
        }
      }
    }
  }
`;

export const PEERS_QUERY = gql`
  query Peers($query: String) {
    peers(query: $query) {
      id
      name
      description
      publicKey
      endpoint
      allowedIPs
      persistentKeepalive
      presharedKey
      createdAt
      updatedAt
      stats {
        endpoint
        lastHandshakeTime
        receiveBytes
        transmitBytes
        protocolVersion
      }
      server {
        id
        name
      }
      backend {
        id
        name
      }
    }
  }
`;

export const PEER_QUERY = gql`
  query Peer($id: ID!) {
    node(id: $id) {
      ... on Peer {
        id
        name
        description
        publicKey
        endpoint
        allowedIPs
        persistentKeepalive
        presharedKey
        createdAt
        updatedAt
        hooks {
          command
          runOnCreate
          runOnDelete
          runOnUpdate
        }
        stats {
          endpoint
          lastHandshakeTime
          receiveBytes
          transmitBytes
          protocolVersion
        }
        server {
          id
          name
        }
        backend {
          id
          name
        }
      }
    }
  }
`;

export const USERS_QUERY = gql`
  query Users($query: String) {
    users(query: $query) {
      id
      email
      createdAt
      updatedAt
    }
  }
`;

export const USER_QUERY = gql`
  query User($id: ID!) {
    node(id: $id) {
      ... on User {
        id
        email
        createdAt
        updatedAt
      }
    }
  }
`;

export const AVAILABLE_BACKENDS_QUERY = gql`
  query AvailableBackends {
    availableBackends {
      type
      supported
      registered
    }
  }
`;

export const BACKENDS_QUERY = gql`
  query Backends($type: String) {
    backends(type: $type) {
      id
      name
      description
      url
      enabled
      supported
      servers {
        id
      }
      createdAt
      updatedAt
    }
  }
`;

export const BACKEND_QUERY = gql`
  query Backend($id: ID!) {
    node(id: $id) {
      ... on Backend {
        id
        name
        description
        url
        enabled
        supported
        servers {
          id
        }
        createdAt
        updatedAt
      }
    }
  }
`;

export const FOREIGN_SERVERS_QUERY = gql`
  query ForeignServers {
    foreignServers {
      name
      type
      publicKey
      listenPort
      firewallMark
      backend {
        id
        name
      }
      foreignInterface {
        name
        addresses
        mtu
      }
      peers {
        publicKey
        endpoint
        allowedIps
        lastHandshakeTime
        persistentKeepAliveInterval
        protocolVersion
        receiveBytes
        transmitBytes
      }
    }
  }
`;
