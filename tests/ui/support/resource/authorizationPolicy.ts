export interface AuthorizationPolicyCommands {
    authorizationPolicySelectAction(action: AuthorizationPolicyAction): void
    authorizationPolicyTypeName(name: string): void
    authorizationPolicyAddRule(rule: AuthorizationPolicyRule): void
    authorizationPolicyAddSelector(key: string, value: string): void
}

export type AuthorizationPolicyAction = "ALLOW" | "DENY" | "AUDIT" | "CUSTOM"

export type AuthorizationPolicyRule = {
    when: {
        key: string;
        value: string;
    };
    to: {
        operation: {
            method: "GET" | "POST" | "PUT" | "DELETE" | "PATCH" | "HEAD" | "OPTIONS" | "CONNECT" | "TRACE";
            path: string;
        };
    };
}

Cypress.Commands.add('authorizationPolicySelectAction', (action: AuthorizationPolicyAction): void => {
    cy.chooseComboboxOption('[data-testid="spec.action"]', action);
});

Cypress.Commands.add('authorizationPolicyTypeName', (name: string): void => {
    cy.inputClearAndType('ui5-input[aria-label="AuthorizationPolicy name"]', name);
});

Cypress.Commands.add('authorizationPolicyAddRule', (rule: AuthorizationPolicyRule): void => {
    cy.addFormGroupItem('[aria-label="expand Rules"]:visible');
    cy.addFormGroupItem('[aria-label="expand When"]:visible');

    cy.inputClearAndType('[data-testid="spec.rules.0.when.0.key"]', rule.when.key);

    cy.get('[aria-label="expand Values"]:visible')
        .click();

    cy.inputClearAndType('[data-testid="spec.rules.0.when.0.values.0"]', rule.when.value);

    cy.addFormGroupItem('[aria-label="expand To"]:visible');

    cy.get('[aria-label="expand Methods"]:visible')
        .click();
    cy.inputClearAndType('[data-testid="spec.rules.0.to.0.operation.methods.0"]', rule.to.operation.method);

    cy.get('[aria-label="expand Paths"]:visible')
        .click();
    cy.inputClearAndType('[data-testid="spec.rules.0.to.0.operation.paths.0"]', rule.to.operation.path);
});


