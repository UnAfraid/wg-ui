import { gql } from "@apollo/client";

export const SERVER_CHANGED_SUBSCRIPTION = gql`
  subscription ServerChanged {
    serverChanged {
      action
      node {
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
  }
`;

export const SERVER_DETAIL_CHANGED_SUBSCRIPTION = gql`
  subscription ServerDetailChanged {
    serverChanged {
      action
      node {
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

export const PEER_CHANGED_SUBSCRIPTION = gql`
  subscription PeerChanged {
    peerChanged {
      action
      node {
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

export const BACKEND_CHANGED_SUBSCRIPTION = gql`
  subscription BackendChanged {
    backendChanged {
      action
      node {
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

export const USER_CHANGED_SUBSCRIPTION = gql`
  subscription UserChanged {
    userChanged {
      action
      node {
        id
        email
        createdAt
        updatedAt
      }
    }
  }
`;
