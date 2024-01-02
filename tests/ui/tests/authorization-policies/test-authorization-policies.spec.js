/// <reference types="cypress" />
import 'cypress-file-upload';
import { chooseComboboxOption } from '../../support/combobox';
import {generateNamespaceName} from "../../support/random";

const AP_NAME =
    'test-ap-' +
    Math.random()
        .toString()
        .substr(2, 8);
const ACTION = 'AUDIT';
const METHODS = 'GET';
const PATHS = '/user/profile/*';
const KEY = 'request.auth.claims[iss]';
const VALUES = 'https://test-value.com';

context('Test Authorization Policies', () => {
    const namespaceName = generateNamespaceName();

    before(() => {
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    after(() => {
        cy.loginAndSelectCluster();
        cy.deleteNamespace(namespaceName);
    });

    it('Create Authorization Policy', () => {
        cy.navigateTo('Istio', 'Authorization Policies');

        cy.contains('ui5-button', 'Create Authorization Policy').click();

        cy.wait(500);

        // Action
        chooseComboboxOption('[placeholder="Type or choose an option."]', ACTION);

        // Name
        cy.get('ui5-dialog')
            .find('[aria-label="AuthorizationPolicy name"]:visible')
            .find('input')
            .type(AP_NAME, { force: true });

        // Rules
        cy.get('[aria-label="expand Rules"]:visible', { log: false })
            .contains('Add')
            .click();

        // When
        cy.get('[aria-label="expand When"]:visible', { log: false })
            .contains('Add')
            .click();

        cy.get('[data-testid="spec.rules.0.when.0.key"]:visible')
            .find('input')
            .type(KEY);

        cy.get('[aria-label="expand Values"]:visible', { log: false }).click();

        cy.get('[data-testid="spec.rules.0.when.0.values.0"]:visible')
            .find('input')
            .type(VALUES);

        // To
        cy.get('[aria-label="expand To"]:visible', { log: false })
            .contains('Add')
            .click();

        cy.get('[aria-label="expand Methods"]:visible', { log: false }).click();

        cy.get('[data-testid="spec.rules.0.to.0.operation.methods.0"]:visible')
            .find('input')
            .type(METHODS);

        cy.get('[aria-label="expand Paths"]:visible', { log: false }).click();

        cy.get('[data-testid="spec.rules.0.to.0.operation.paths.0"]:visible')
            .find('input')
            .type(PATHS);

        cy.get('ui5-dialog')
            .contains('ui5-button', 'Create')
            .should('be.visible')
            .click();
    });

    it('Checking details', () => {
        cy.contains(AP_NAME).should('be.visible');

        cy.contains(ACTION).should('be.visible');

        cy.contains('Matches all Pods in the Namespace').should('be.visible');

        cy.contains('Rule #1 to when', { timeout: 10000 }).click();

        cy.contains('To #1 methods paths', { timeout: 10000 }).click();

        cy.contains(PATHS).should('be.visible');

        cy.contains(KEY).should('be.visible');

        cy.contains(VALUES).should('be.visible');

        cy.contains('Operation').should('be.visible');

        cy.contains(METHODS).should('be.visible');
    });

    it('Edit and check changes', () => {
        cy.contains('ui5-button', 'Edit').click();

        cy.get('[placeholder="Enter key"]:visible', { log: false })
            .find('input')
            .filterWithNoValue()
            .type('sel', { force: true });

        cy.get('[placeholder="Enter value"]:visible', { log: false })
            .find('input')
            .filterWithNoValue()
            .first()
            .type('selector-value', { force: true });

        cy.get('ui5-dialog')
            .contains('ui5-button', 'Update')
            .should('be.visible')
            .click();

        cy.contains('sel=selector-value').should('be.visible');

        cy.contains('Matches all Pods in the Namespace').should('not.exist');
    });

    it('Inspect list', () => {
        cy.inspectList('Authorization Policies', 'test-ap');
    });
});
