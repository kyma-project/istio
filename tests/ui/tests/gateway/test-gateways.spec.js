/// <reference types="cypress" />
import 'cypress-file-upload';
import {chooseComboboxOption} from '../../support/combobox';
import {generateNamespaceName, generateRandomName} from "../../support/random";

const GATEWAY_NAME = generateRandomName('test-gateway')

const SERVER_NAME = GATEWAY_NAME + '-server';
const PORT_NUMBER = 80;
const PORT_PROTOCOL = 'HTTP';
const SELECTOR = 'selector=selector-value';

const KYMA_GATEWAY_CERTS = 'kyma-gateway-certs';

context('Test Gateways', () => {
    const namespaceName = generateNamespaceName();
    const serviceName = generateRandomName("test-service");

    before(() => {
        cy.loginAndSelectCluster();
        cy.createNamespace(namespaceName);
        cy.createService(serviceName);
    });

    after(() => {
        cy.loginAndSelectCluster();
        cy.deleteNamespace(namespaceName);
    });

    it('Create Gateway', () => {
        cy.navigateTo('Istio', 'Gateways');

        cy.clickCreateButton();

        // name
        cy.get('ui5-dialog')
            .find('[aria-label="Gateway name"]:visible')
            .find('input')
            .type(GATEWAY_NAME, {force: true});

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

        // server
        cy.get('[aria-label="expand Servers"]:visible', {log: false})
            .contains('Add')
            .click();

        cy.get('[data-testid="spec.servers.0.port.number"]:visible')
            .find('input')
            .type(PORT_NUMBER);

        chooseComboboxOption(
            '[data-testid="spec.servers.0.port.protocol"]',
            PORT_PROTOCOL,
        );

        cy.get('[aria-label^="Gateway name"]:visible', {log: false})
            .find('input')
            .eq(1)
            .type(SERVER_NAME, {force: true});

        // hosts
        cy.get('[aria-label="expand Hosts"]:visible', {log: false}).click();

        cy.get('[placeholder="For example, *.api.mydomain.com"]:visible', {
            log: false,
        })
            .find('input')
            .type('example.com');

        cy.get('[placeholder="For example, *.api.mydomain.com"]:visible', {
            log: false,
        })
            .find('input')
            .filterWithNoValue()
            .type('*.example.com', {force: true});

        // create
        cy.get('ui5-dialog')
            .contains('ui5-button', 'Create')
            .should('be.visible')
            .click();
    });

    it('Inspect details', () => {
        cy.contains(GATEWAY_NAME);
        cy.contains(SELECTOR);
        // default selector
        cy.contains('istio=ingressgateway');
        cy.contains(SERVER_NAME);
        cy.contains(PORT_NUMBER);
        // hosts
        cy.contains('example.com');
        cy.contains('*.example.com');
    });

    it('Edit Gateway', () => {
        cy.contains('ui5-button', 'Edit').click();

        cy.get('ui5-dialog')
            .find('[aria-label="Gateway name"]:visible')
            .find('input')
            .should('have.attr', 'readonly');

        cy.get('[aria-label="expand Servers"]:visible', {
            log: false,
        }).click();

        // change server to HTTPS
        cy.get(`ui5-combobox[data-testid="spec.servers.0.port.protocol"]`)
            .find('input')
            .click()
            .clear({force: true})
            .type('HTTPS');

        cy.get('ui5-li:visible')
            .contains('HTTPS')
            .find('li')
            .click({force: true});

        cy.get('[data-testid="spec.servers.0.port.number"]:visible')
            .find('input')
            .click()
            .clear({force: true})
            .type('443');

        cy.get('[aria-label="expand Port"]:visible', {
            log: false,
        }).click();

        cy.get('[aria-label="expand TLS"]:visible', {
            log: false,
        }).click();

        // secret
        cy.get('[aria-label="Choose Secret"]:visible', {
            log: false,
        })
            .find('input')
            .type(KYMA_GATEWAY_CERTS, {force: true});

        chooseComboboxOption('[data-testid="spec.servers.0.tls.mode"]', 'SIMPLE');

        cy.get('ui5-dialog')
            .contains('ui5-button', 'Update')
            .should('be.visible')
            .click();

        // changed details
        cy.contains('443');
        cy.contains('HTTPS');
        cy.contains(/simple/i);
        cy.contains(KYMA_GATEWAY_CERTS);
    });

    it('Inspect list', () => {
        cy.navigateBackTo('Gateways', GATEWAY_NAME);
        cy.checkItemOnGenericListLink(GATEWAY_NAME);
    });
});
