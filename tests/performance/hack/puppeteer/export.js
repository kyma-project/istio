import puppeteer from 'puppeteer';

(async () => {
    // Launch the browser and open a new blank page
    const browser = await puppeteer.launch();
    const page = await browser.newPage();
    await page.setExtraHTTPHeaders({
        'Authorization': 'Basic YWRtaW46YWRtaW4='
    });


    // Navigate the page to a URL
    await page.goto('https://grafana.bc-perf.goatz.shoot.canary.k8s-hana.ondemand.com/d/XwO4kRSnz/istio-performance?orgId=1');

    // Set screen size
    await page.setViewport({width: 1080, height: 1024});

    // Type into the search box
    const buttonSelector = 'button[class="css-orvko6"]';
    await page.waitForSelector(buttonSelector);
    await page.$$eval(buttonSelector, el => el[1].click());

    // Wait and click on the first result
    const liSelector = 'li[aria-label="Tab Snapshot"]';
    await page.waitForSelector(liSelector);
    await page.click(liSelector);

    const publishSelector = "::-p-text(Publish to snapshot.raintank.io)";
    await page.waitForSelector(publishSelector);
    await page.click(publishSelector)

    const linkSelector = 'a[class="large share-modal-link"]'
    await page.waitForSelector(linkSelector);
    let link = await page.$(linkSelector)
    let linkValue = await page.evaluate(el => el.textContent, link)
    // Print the full title
    console.log(linkValue);

    await browser.close();
})();
