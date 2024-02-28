/// <reference types="cypress" />
import 'cypress-file-upload';
import {generateNamespaceName, generateRandomName} from "../../support/random";

const DR_NAME = generateRandomName('test-dr')
const HOST = 'ratings.prod.svc.cluster.local';
const SELECTOR = 'selector=selector-value';

context('Test Destination Rules', () => {
    const namespaceName = generateNamespaceName();

    before(() => {
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    after(() => {
        cy.loginAndSelectCluster();
        cy.deleteNamespace(namespaceName);
    });

    it('Create a Destination Rule', () => {
        cy.navigateTo('Istio', 'Destination Rules');

        cy.clickCreateButton();

        cy.get('ui5-dialog')
            .find('[aria-label="DestinationRule name"]:visible')
            .find('input')
            .type(DR_NAME, {force: true});

        cy.get('[data-testid="spec.host"]:visible')
            .find('input')
            .click()
            .type(HOST, {force: true});

        cy.get('ui5-dialog')
            .contains('ui5-button', 'Create')
            .should('be.visible')
            .click();

        cy.contains('ui5-title', DR_NAME).should('be.visible');
    });

    it('Check Destination Rule details', () => {
        cy.contains(HOST).should('be.visible');

        cy.contains('Subsets').should('not.exist');

        cy.contains('Workload Selector').should('not.exist');
    });

    it('Edit Destination Rule', () => {
        cy.contains('ui5-button', 'Edit').click();

        cy.get('ui5-dialog')
            .find('[aria-label="DestinationRule name"]:visible')
            .find('input')
            .should('have.attr', 'readonly');

        // selector
        cy.get('[placeholder="Enter key"]:visible', {log: false})
            .find('input')
            .filterWithNoValue()
            .type('selector', {force: true});

        cy.get('[placeholder="Enter value"]:visible', {log: false})
            .find('input')
            .filterWithNoValue()
            .first()
            .type('selector-value', {force: true});

        cy.get('ui5-dialog')
            .contains('ui5-button', 'Update')
            .should('be.visible')
            .click();

        // changed details
        cy.contains(SELECTOR);
        // After resolving: https://github.com/kyma-project/busola/issues/2088 we need to add checking loadBalancer value
    });

    it('Check the Destination Rule list', () => {
        cy.navigateBackTo('Destination Rules', DR_NAME);
        cy.checkItemOnGenericListLink(DR_NAME);
    });
});
