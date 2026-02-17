import { gql } from "@apollo/client";

export const SIGN_IN_MUTATION = gql`
  mutation SignIn($input: SignInInput!) {
    signIn(input: $input) {
      token
      expiresAt
      expiresIn
    }
  }
`;

export const CREATE_SERVER_MUTATION = gql`
  mutation CreateServer($input: CreateServerInput!) {
    createServer(input: $input) {
      server {
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
      }
    }
  }
`;

export const UPDATE_SERVER_MUTATION = gql`
  mutation UpdateServer($input: UpdateServerInput!) {
    updateServer(input: $input) {
      server {
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
        hooks {
          command
          runOnCreate
          runOnDelete
          runOnStart
          runOnStop
          runOnUpdate
        }
        updatedAt
      }
    }
  }
`;

export const DELETE_SERVER_MUTATION = gql`
  mutation DeleteServer($input: DeleteServerInput!) {
    deleteServer(input: $input) {
      server {
        id
      }
    }
  }
`;

export const START_SERVER_MUTATION = gql`
  mutation StartServer($input: StartServerInput!) {
    startServer(input: $input) {
      server {
        id
        running
        interfaceStats {
          rxBytes
          txBytes
        }
      }
    }
  }
`;

export const STOP_SERVER_MUTATION = gql`
  mutation StopServer($input: StopServerInput!) {
    stopServer(input: $input) {
      server {
        id
        running
      }
    }
  }
`;

export const CREATE_PEER_MUTATION = gql`
  mutation CreatePeer($input: CreatePeerInput!) {
    createPeer(input: $input) {
      peer {
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
        server {
          id
          name
        }
      }
    }
  }
`;

export const UPDATE_PEER_MUTATION = gql`
  mutation UpdatePeer($input: UpdatePeerInput!) {
    updatePeer(input: $input) {
      peer {
        id
        name
        description
        publicKey
        endpoint
        allowedIPs
        persistentKeepalive
        presharedKey
        hooks {
          command
          runOnCreate
          runOnDelete
          runOnUpdate
        }
        updatedAt
      }
    }
  }
`;

export const DELETE_PEER_MUTATION = gql`
  mutation DeletePeer($input: DeletePeerInput!) {
    deletePeer(input: $input) {
      peer {
        id
      }
    }
  }
`;

export const CREATE_USER_MUTATION = gql`
  mutation CreateUser($input: CreateUserInput!) {
    createUser(input: $input) {
      user {
        id
        email
        createdAt
        updatedAt
      }
    }
  }
`;

export const UPDATE_USER_MUTATION = gql`
  mutation UpdateUser($input: UpdateUserInput!) {
    updateUser(input: $input) {
      user {
        id
        email
        updatedAt
      }
    }
  }
`;

export const DELETE_USER_MUTATION = gql`
  mutation DeleteUser($input: DeleteUserInput!) {
    deleteUser(input: $input) {
      user {
        id
      }
    }
  }
`;

export const CREATE_BACKEND_MUTATION = gql`
  mutation CreateBackend($input: CreateBackendInput!) {
    createBackend(input: $input) {
      backend {
        id
        name
        description
        url
        enabled
        supported
        createdAt
        updatedAt
      }
    }
  }
`;

export const UPDATE_BACKEND_MUTATION = gql`
  mutation UpdateBackend($input: UpdateBackendInput!) {
    updateBackend(input: $input) {
      backend {
        id
        name
        description
        url
        enabled
        supported
        updatedAt
      }
    }
  }
`;

export const DELETE_BACKEND_MUTATION = gql`
  mutation DeleteBackend($input: DeleteBackendInput!) {
    deleteBackend(input: $input) {
      backend {
        id
      }
    }
  }
`;

export const GENERATE_WIREGUARD_KEY_MUTATION = gql`
  mutation GenerateWireguardKey($input: GenerateWireguardKeyInput!) {
    generateWireguardKey(input: $input) {
      privateKey
      publicKey
    }
  }
`;

export const IMPORT_FOREIGN_SERVER_MUTATION = gql`
  mutation ImportForeignServer($input: ImportForeignServerInput!) {
    importForeignServer(input: $input) {
      server {
        id
        name
        address
        listenPort
        running
        publicKey
        backend {
          id
          name
        }
      }
    }
  }
`;
