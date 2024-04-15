export interface VirtualServiceCommands {
    virtualServiceTypeName(value: string): void
    virtualServiceAddHttpRoute(server: VirtualServiceHttpRoute): void
}

type VirtualServiceHttpRoute = {
    matchName: string;
    uri: {
        prefix: "exact" | "prefix" | "regex";
        value: string;
    },
    redirect: {
        uri: string;
        authority: string;
    }
}

Cypress.Commands.add('virtualServiceTypeName', (value: string): void => {
    cy.inputClearAndType('ui5-input[aria-label="VirtualService name"]', value);
});

Cypress.Commands.add('virtualServiceAddHttpRoute', (route: VirtualServiceHttpRoute): void => {
    cy.addFormGroupItem('[aria-label="expand HTTP"]:visible');
    cy.addFormGroupItem('[aria-label="expand Matches"]:visible');

    cy.inputClearAndType('[data-testid="spec.http.0.match.0.name"]', route.matchName);

    // URI
    cy.get('[aria-label="expand URI"]:visible')
        .click();

    cy.chooseComboboxFixedOption('[data-testid="select-dropdown"]', route.uri.prefix);

    cy.get('[placeholder="Enter value"]:visible')
        .find('input')
        .filterWithNoValue()
        .first()
        .type(route.uri.value, {force: true});

    // Redirect
    cy.get('[aria-label="expand Redirect"]')
        .first()
        .click();

    cy.inputClearAndType('[data-testid="spec.http.0.redirect.uri"]', route.redirect.uri);
    cy.inputClearAndType('[data-testid="spec.http.0.redirect.authority"]', route.redirect.authority);

});
