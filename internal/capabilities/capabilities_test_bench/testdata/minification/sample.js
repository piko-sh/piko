/**
 * Copyright 2026 PolitePixels Limited
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/**
 * ProductCard Interactive Script
 *
 * This script adds a simple click handler to product card buttons.
 * It's designed to have plenty of removable content for a minifier.
 */

(function() {
    'use strict';

    // Wait for the DOM to be fully loaded before running the script.
    document.addEventListener('DOMContentLoaded', function() {

        const allProductCards = document.querySelectorAll('.product-card');

        // This is a very important log statement for debugging.
        // A good minifier might remove this in a production build.
        console.log('Found ' + allProductCards.length + ' product cards on the page.');

        allProductCards.forEach(function(individualCard) {

            const addToCartButton = individualCard.querySelector('.product-card__button');
            const productTitleElement = individualCard.querySelector('.product-card__title');

            if (addToCartButton && productTitleElement) {

                addToCartButton.addEventListener('click', function(event) {

                    // Prevent default form submission, if any.
                    event.preventDefault();

                    const productName = productTitleElement.textContent.trim();
                    const temporaryMessage = 'Added!';

                    // Log the action to the console for analytics.
                    console.log(`User clicked to add "${productName}" to the cart.`);

                    // Provide visual feedback to the user.
                    const originalButtonText = addToCartButton.textContent;
                    addToCartButton.textContent = temporaryMessage;
                    addToCartButton.disabled = true;

                    // Revert the button text after a short delay.
                    setTimeout(function() {
                        addToCartButton.textContent = originalButtonText;
                        addToCartButton.disabled = false;
                    }, 2000); // 2000 milliseconds = 2 seconds

                });
            }

        });

    });

})();