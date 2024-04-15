import {loadFixture} from "./loadFile";
import {postApis} from "./httpClient";

type AuthorizationPolicy = {
    apiVersion: string;
    metadata: {
        name: string;
        namespace: string;
    }
}

Cypress.Commands.add('createAuthorizationPolicy', (namespace: string, name: string) => {
    // @ts-ignore Typing of cy.then is not good enough
    cy.wrap(loadFixture('authorizationPolicy.yaml')).then((a: AuthorizationPolicy): void => {
        a.metadata.name = name;
        a.metadata.namespace = namespace;

        // We have to use cy.wrap, since the post command uses a cy.fixture internally
        cy.wrap(postApis(`${a.apiVersion}/namespaces/${namespace}/authorizationpolicies`, a)).should("be.true");
    })
});