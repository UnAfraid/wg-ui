import {
  ApolloClient,
  InMemoryCache,
  HttpLink,
  split,
  from,
} from "@apollo/client";
import { GraphQLWsLink } from "@apollo/client/link/subscriptions";
import { getMainDefinition } from "@apollo/client/utilities";
import { setContext } from "@apollo/client/link/context";
import { onError } from "@apollo/client/link/error";
import { createClient } from "graphql-ws";
import { getToken } from "@/lib/auth";

const GRAPHQL_URL =
  import.meta.env.VITE_GRAPHQL_URL || "/query";

function getWsUrl() {
  const url = new URL(GRAPHQL_URL, window.location.origin);
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}

const httpLink = new HttpLink({
  uri: (operation) => {
    const name = operation.operationName;
    return name ? `${GRAPHQL_URL}?name=${name}` : GRAPHQL_URL;
  },
});

const authLink = setContext((_, { headers }) => {
  const token = getToken();
  return {
    headers: {
      ...headers,
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  };
});

const errorLink = onError(({ graphQLErrors, networkError }) => {
  if (graphQLErrors) {
    graphQLErrors.forEach(({ message, locations, path }) => {
      console.error(
        `[GraphQL error]: Message: ${message}, Location: ${JSON.stringify(locations)}, Path: ${path}`
      );
    });
  }
  if (networkError) {
    console.error(`[Network error]: ${networkError}`);
  }
});

function createWsLink() {
  const wsUrl = getWsUrl();
  return new GraphQLWsLink(
    createClient({
      url: `${wsUrl}${wsUrl.includes("?") ? "&" : "?"}type=subscription`,
      connectionParams: () => {
        const token = getToken();
        return token ? { Authorization: `Bearer ${token}` } : {};
      },
      shouldRetry: () => true,
      retryAttempts: Infinity,
      retryWait: async (retries) => {
        const delay = Math.min(1000 * 2 ** retries, 30000);
        await new Promise((resolve) => setTimeout(resolve, delay));
      },
    })
  );
}

function createLink() {
  const httpChain = from([authLink, errorLink, httpLink]);
  const wsLink = createWsLink();

  return split(
    ({ query }) => {
      const definition = getMainDefinition(query);
      return (
        definition.kind === "OperationDefinition" &&
        definition.operation === "subscription"
      );
    },
    wsLink,
    httpChain
  );
}

let _client: ApolloClient<unknown> | null = null;

export function makeApolloClient() {
  if (!_client) {
    _client = new ApolloClient({
      link: createLink(),
      cache: new InMemoryCache(),
      defaultOptions: {
        watchQuery: {
          fetchPolicy: "cache-and-network",
        },
      },
      ssrMode: false,
    });
  }
  return _client;
}

export const apolloClient = makeApolloClient();
