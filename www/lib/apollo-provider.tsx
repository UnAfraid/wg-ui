"use client";

import { useMemo } from "react";
import { ApolloProvider } from "@apollo/client";
import { makeApolloClient } from "./apollo-client";

export function ApolloWrapper({ children }: { children: React.ReactNode }) {
  const client = useMemo(() => makeApolloClient(), []);
  return <ApolloProvider client={client}>{children}</ApolloProvider>;
}
