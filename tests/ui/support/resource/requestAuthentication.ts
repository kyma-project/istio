export interface RequestAuthenticationCommands {
    requestAuthenticationTypeName(name: string): void
    requestAuthenticationAddJwtRule(rule: JwtRule, numberOfJwtRule?: number): void
    requestAuthenticationAddJwtRules(rules: JwtRule[]): void
}

export type JwtRule = {
    issuer: string;
    jwksUri: string;
    audiences: string[];
    fromParams: string[];
    fromCookies: string[];
    fromHeaders: FromHeaders[];
}

export type FromHeaders = {
    name: string;
    prefix: string;
}

Cypress.Commands.add('requestAuthenticationTypeName', (name: string): void => {
    cy.inputClearAndType('ui5-input[aria-label="RequestAuthentication name"]', name);
});

Cypress.Commands.add('requestAuthenticationAddJwtRules', (rules: JwtRule[]): void => {
   rules.forEach((jwt, index) => {
         cy.requestAuthenticationAddJwtRule(jwt, index);
   });
});

Cypress.Commands.add('requestAuthenticationAddJwtRule', (rule: JwtRule, numberOfJwtRule= 0): void => {
    cy.addFormGroupItem('[aria-label="expand JWT Rules"]:visible');
    cy.inputClearAndType(`ui5-input[data-testid="spec.jwtRules.${numberOfJwtRule}.issuer"]`, rule.issuer);
    cy.inputClearAndType(`ui5-input[data-testid="spec.jwtRules.${numberOfJwtRule}.jwksUri"]`, rule.jwksUri);

    cy.get('[aria-label="expand Audiences"]:visible').click();
    rule.audiences.forEach((audience, index) => {
        cy.inputClearAndType(`ui5-input[data-testid="spec.jwtRules.${numberOfJwtRule}.audiences.${index}"]`, audience);
    });

    cy.get('[aria-label="expand From Params"]:visible').click();
    rule.fromParams.forEach((param, index) => {
        cy.inputClearAndType(`ui5-input[data-testid="spec.jwtRules.${numberOfJwtRule}.fromParams.${index}"]`, param);
    });

    cy.get('[aria-label="expand From Cookies"]:visible').click();
    rule.fromCookies.forEach((cookie, index) => {
        cy.inputClearAndType(`ui5-input[data-testid="spec.jwtRules.${numberOfJwtRule}.fromCookies.${index}"]`, cookie);
    });

    rule.fromHeaders.forEach((header, index) => {
        cy.addFormGroupItem(`[aria-label="expand From Headers"]:visible`);
        cy.inputClearAndType(`ui5-input[data-testid="spec.jwtRules.${numberOfJwtRule}.fromHeaders.${index}.name"]`, header.name);
        cy.inputClearAndType(`ui5-input[data-testid="spec.jwtRules.${numberOfJwtRule}.fromHeaders.${index}.prefix"]`, header.prefix);
    });

    cy.get('[aria-label="expand JWT Rule"]:visible').eq(numberOfJwtRule).click();
});
