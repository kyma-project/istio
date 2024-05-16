import {loadFixture} from "./loadFile";
import * as k8s from "@kubernetes/client-node";
import {postApi} from "./httpClient";

Cypress.Commands.add('createHttpbinSleepPod', (name: string, namespace: string) => {
  // @ts-ignore Typing of cy.then is not good enough
  cy.wrap(loadFixture('pod-httpbin-sleep.yaml')).then((s: k8s.V1Pod): void => {
    s.metadata!.name = name
    s.metadata!.namespace = namespace
    s.metadata!.labels ? s.metadata!.labels["app"] = name : s.metadata!.labels = {"app": name}
    // We have to use cy.wrap, since the post command uses a cy.fixture internally
    cy.wrap(postApi(`v1/namespaces/${namespace}/pods`, s)).should("be.true");
  })
});


