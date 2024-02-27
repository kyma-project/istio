/// <reference types="cypress" />
import 'cypress-file-upload';
import {chooseComboboxOption} from '../../support/combobox';
import {generateNamespaceName, generateRandomName} from "../../support/random";

const SE_NAME = generateRandomName('test')
const RESOLUTION = 'STATIC';
const LOCATION = 'MESH_EXTERNAL';
const HOST = 'test.com';
const ADDRESS = '192.192.192.192/24';
const WORKLOAD_SELECTOR_LABEL = 'test=selector-value';

context('Test Service Entries', () => {
    const namespaceName = generateNamespaceName();

    before(() => {
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
    });

    after(() => {
        cy.loginAndSelectCluster();
        cy.deleteNamespace(namespaceName);
    });

    it('Create a Service Entry', () => {
        cy.navigateTo('Istio', 'Service Entries');

        cy.clickCreateButton();

        cy.contains('Advanced').click();

        // Name
        cy.get('ui5-dialog')
            .find('[aria-label="ServiceEntry name"]:visible')
            .find('input')
            .click()
            .type(SE_NAME, {force: true});

        // Hosts
        cy.get('[aria-label="expand Hosts"]:visible', {log: false}).click();

        cy.get('[data-testid="spec.hosts.0"]:visible')
            .find('input')
            .type(HOST, {force: true});

        chooseComboboxOption('[data-testid="spec.resolution"]', RESOLUTION);

        chooseComboboxOption('[data-testid="spec.location"]', LOCATION);

        cy.get('[aria-label="expand Addresses"]:visible', {log: false}).click();

        cy.get('[placeholder="For example, 127.0.0.1"]:visible', {
            log: false,
        })
            .find('input')
            .type(ADDRESS, {force: true});

        cy.get('[placeholder="Enter key"]:visible', {log: false})
            .find('input')
            .filterWithNoValue()
            .type('test', {force: true});

        cy.get('[placeholder="Enter value"]:visible', {log: false})
            .find('input')
            .filterWithNoValue()
            .first()
            .type('selector-value', {force: true});

        cy.get('ui5-dialog')
            .contains('ui5-button', 'Create')
            .should('be.visible')
            .click();

        cy.contains('ui5-title', SE_NAME).should('be.visible');
    });

    it('Check the Service Entry details', () => {
        cy.get('#content-wrap')
            .should('include.text', RESOLUTION)
            .and('include.text', LOCATION)
            .and('include.text', HOST)
            .and('include.text', ADDRESS);

        cy.contains(WORKLOAD_SELECTOR_LABEL);
    });

    it('Check the Service Entries list', () => {
        cy.navigateBackTo('Service Entries', SE_NAME);
        cy.checkItemOnGenericListLink(SE_NAME);
        cy.contains(RESOLUTION);
    });
});
