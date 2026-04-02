// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package kitchen

// Event is a single kitchen activity notification.
type Event struct {
	Text     string
	Category string // "mouse", "cheese", "alarm", "chef", "review", "delivery", "mystery"
}

// Sequence is an ordered chain of events that play consecutively.
type Sequence []Event

// Sequences is the pool of multi-part event chains. Each chain plays
// in order, with standalone events interleaved between them.
var Sequences = []Sequence{
	// Gerald the Mouse
	{
		{Text: "Mouse spotted near the Gruyere. Security has been alerted.", Category: "mouse"},
		{Text: "UPDATE: Mouse has been identified as Gerald. He works here.", Category: "mouse"},
		{Text: "Gerald has eaten 200g of Gruyere. HR has been notified.", Category: "mouse"},
		{Text: "Gerald has been promoted to Head of Quality Control. The Gruyere was excellent.", Category: "mouse"},
	},
	// The Camembert Podcast
	{
		{Text: "ALERT: The walk-in Camembert has reached critical ripeness.", Category: "cheese"},
		{Text: "UPDATE: The Camembert has started a podcast. Episode 1: 'Why I'm Better Than Brie.'", Category: "cheese"},
		{Text: "The Camembert podcast now has 4,000 subscribers. We may need a bigger walk-in.", Category: "cheese"},
	},
	// Smoke Alarm Saga
	{
		{Text: "Smoke alarm triggered in Kitchen 2. Chef insists this is 'ambiance.'", Category: "alarm"},
		{Text: "Fire brigade has arrived. Chef is offering them cheese toasties.", Category: "alarm"},
		{Text: "Fire brigade has left. They took two toasties each and gave us a 5-star review.", Category: "review"},
	},
	// The Great Parmesan Escape
	{
		{Text: "A wheel of parmesan has rolled off the counter. It is currently heading for the exit.", Category: "cheese"},
		{Text: "UPDATE: The parmesan has made it to the dining room. Table 4 is cheering it on.", Category: "cheese"},
		{Text: "The parmesan has been captured under the dessert trolley. It put up a good fight.", Category: "cheese"},
	},
	// The Delivery Driver
	{
		{Text: "Delivery driver refused to leave until he tried the mac and cheese. He's still here.", Category: "delivery"},
		{Text: "UPDATE: Delivery driver has been here for 45 minutes. He has ordered seconds.", Category: "delivery"},
		{Text: "The delivery driver has applied for a job. His van is still running outside.", Category: "delivery"},
	},
	// Fondue Incident
	{
		{Text: "Table 9 has declared fondue independence. They refuse to share the pot.", Category: "cheese"},
		{Text: "Negotiations with Table 9 have broken down. They have built a bread wall around the fondue.", Category: "cheese"},
		{Text: "Table 9 has agreed to a ceasefire. Terms: unlimited bread supply and a second pot.", Category: "cheese"},
	},
	// Mouse Council
	{
		{Text: "The mice have called an emergency meeting in the pantry. Agenda unknown.", Category: "mouse"},
		{Text: "UPDATE: The mice have voted. They want Fridays to be Brie Day.", Category: "mouse"},
		{Text: "Management has accepted the mice's proposal. Brie Fridays begin next week.", Category: "mouse"},
	},
	// The Health Inspector (long)
	{
		{Text: "ALERT: Health inspector has arrived unannounced. All stations to battle positions.", Category: "alarm"},
		{Text: "Gerald has been told to hide. Gerald is refusing to hide. Gerald says this is his kitchen too.", Category: "mouse"},
		{Text: "UPDATE: Gerald is now sitting on the inspector's clipboard. The inspector has not noticed.", Category: "mouse"},
		{Text: "The inspector has asked about the small pawprints on the Gruyere. Chef said it's 'artisanal texturing.'", Category: "chef"},
		{Text: "Inspector is now in the cheese cave. He has been in there for 20 minutes. We are concerned.", Category: "alarm"},
		{Text: "UPDATE: The inspector has emerged from the cheese cave. He is weeping. He says it's the most beautiful thing he's ever seen.", Category: "review"},
		{Text: "We have passed inspection. The inspector has asked if we're hiring. Gerald is showing him around.", Category: "mouse"},
	},
	// The Cheddar Shortage Crisis (long)
	{
		{Text: "CRITICAL: Cheddar reserves have dropped below emergency levels. Kitchen-wide rationing in effect.", Category: "cheese"},
		{Text: "Chef has locked the remaining cheddar in the safe. Only he knows the combination.", Category: "chef"},
		{Text: "UPDATE: Someone has cracked the safe. 300g of cheddar is missing. Crumbs lead to Kitchen 3.", Category: "mystery"},
		{Text: "Interrogations underway. The sous chef's alibi is 'I was grating parmesan.' No one believes him.", Category: "chef"},
		{Text: "WITNESS: Gerald reports seeing a human-shaped figure sneaking cheddar at 03:00. He would like a reward.", Category: "mouse"},
		{Text: "The mystery thief has been identified. It was the dishwasher. They said the cheddar 'called to them.'", Category: "mystery"},
		{Text: "Emergency cheddar delivery has arrived. Staff are applauding. The dishwasher has been forgiven.", Category: "delivery"},
	},
	// The Raclette Grill Incident (long)
	{
		{Text: "Someone has plugged in all four raclette grills simultaneously. The lights are flickering.", Category: "alarm"},
		{Text: "UPDATE: The fuse has blown in Kitchen 1. The raclette grills continue to operate on what we can only describe as 'cheese willpower.'", Category: "mystery"},
		{Text: "An electrician has been called. He says he's never seen anything like this. The cheese is somehow generating its own heat.", Category: "mystery"},
		{Text: "The electrician has abandoned the repair and is now eating raclette. He says he'll fix it 'after this next slice.'", Category: "chef"},
		{Text: "Three hours later. The electrician is still eating raclette. His office has called twice. He's not going back.", Category: "review"},
	},
	// Gerald's Day Off
	{
		{Text: "Gerald has requested a personal day. Reason given: 'cheese hangover.'", Category: "mouse"},
		{Text: "Without Gerald, the cheese cave has descended into anarchy. Nobody knows where the Comte is.", Category: "mystery"},
		{Text: "A junior mouse named Beatrice has attempted to fill Gerald's role. She put the Brie in the cheddar section. Pandemonium.", Category: "mouse"},
		{Text: "Gerald has been called back from his day off. He has arrived wearing a tiny bathrobe and is furious.", Category: "mouse"},
		{Text: "Order has been restored. Gerald has reorganised the entire cave in 12 minutes. Beatrice has been reassigned to bread.", Category: "mouse"},
	},
	// The Food Critic
	{
		{Text: "ALERT: A food critic has been spotted at Table 6. She is taking notes. Everyone look natural.", Category: "review"},
		{Text: "The critic has ordered the cheese board. Chef is personally selecting each piece. His hands are shaking.", Category: "chef"},
		{Text: "UPDATE: The critic has tasted the aged Gouda. She closed her eyes for eleven seconds. We are analysing this.", Category: "review"},
		{Text: "The critic asked to speak to the chef. He is in the toilets practising his 'relaxed and confident' face.", Category: "chef"},
		{Text: "The critic told chef his Camembert was 'adequate.' Chef has locked himself in the walk-in to process this.", Category: "chef"},
		{Text: "CORRECTION: The critic meant 'exquisite.' She has a cold and her pronunciation is off. Chef has emerged from the walk-in. He is crying happy tears.", Category: "review"},
	},
	// The Wine Pairing Argument
	{
		{Text: "A heated debate has erupted between Kitchen 1 and Kitchen 2 over whether Merlot or Pinot goes with Gruyere.", Category: "chef"},
		{Text: "The argument has escalated. Kitchen 1 has barricaded the door with a cheese trolley.", Category: "alarm"},
		{Text: "Gerald has been sent in as a neutral mediator. He has eaten the Gruyere in question. Problem solved.", Category: "mouse"},
	},
	// The Missing Mozzarella
	{
		{Text: "40kg of mozzarella delivered this morning has vanished. The walk-in is empty. The delivery note is there. The cheese is not.", Category: "delivery"},
		{Text: "CCTV review shows the mozzarella being wheeled out the back door at 11:47 by a figure in a chef's hat. We have 14 chefs.", Category: "mystery"},
		{Text: "All 14 chefs have been lined up. All 14 are wearing their hats. None of them can explain why one hat smells like mozzarella.", Category: "chef"},
		{Text: "The mozzarella has been found in Kitchen 3's oven. All 40kg. In one batch. Someone was attempting the world's largest pizza.", Category: "mystery"},
		{Text: "The world's largest pizza attempt has been abandoned. But not before we took a photo. It's now the screensaver on every terminal.", Category: "review"},
	},
	// The Cheese Cave Tour
	{
		{Text: "A group of tourists has somehow found the cheese cave and are taking selfies with the Roquefort.", Category: "review"},
		{Text: "Gerald is now giving the tourists an unauthorised tour. He is standing on a wheel of Brie and gesturing importantly.", Category: "mouse"},
		{Text: "The tourists have started an impromptu cheese tasting. They are rating each cheese out of ten. The Stilton is winning.", Category: "review"},
		{Text: "Security has arrived to escort the tourists out. The tourists have refused. They say they live here now.", Category: "mystery"},
		{Text: "A compromise has been reached. The tourists may stay for lunch. They have ordered seven cheese boards between four people.", Category: "review"},
	},
	// The Gruyere Heist
	{
		{Text: "ALERT: 12kg of aged Gruyere has gone missing from the vault. This is not a metaphor. We have an actual vault.", Category: "alarm"},
		{Text: "Gerald has assembled a crack team of mice investigators. They are wearing tiny magnifying glasses. We did not provide these.", Category: "mouse"},
		{Text: "The investigation has uncovered a tunnel behind the walk-in fridge. It leads to the pub next door.", Category: "mystery"},
		{Text: "The pub landlord has been confronted. He claims the Gruyere 'walked in on its own.' He is serving it on toast.", Category: "mystery"},
		{Text: "A treaty has been signed. The pub will return 6kg and keep 6kg. Diplomatic relations have been established. Gerald is ambassador.", Category: "mouse"},
	},
	// The Cheese Wheel Race
	{
		{Text: "Kitchen staff have organised an unofficial cheese wheel race down the corridor. Bets are being placed.", Category: "chef"},
		{Text: "UPDATE: The Gouda is in the lead. The Brie has gone off-course and is heading for the dining room.", Category: "cheese"},
		{Text: "The Brie has taken out a waiter. He is fine. The Brie is not. A moment of silence.", Category: "cheese"},
		{Text: "The Gouda has won. Prize: it will not be eaten today. The Cheddar came last and has been sentenced to a toastie.", Category: "chef"},
	},
	// Gerald's Performance Review
	{
		{Text: "Gerald's annual performance review is today. He has prepared a PowerPoint. We do not know how.", Category: "mouse"},
		{Text: "Gerald's presentation is titled 'Why I Deserve More Gruyere: A Data-Driven Analysis.' Slide 1 is just a photo of himself looking noble.", Category: "mouse"},
		{Text: "UPDATE: Gerald has presented 47 slides. Management is weeping. Not from emotion - from the sheer volume of cheese consumption data.", Category: "mouse"},
		{Text: "Gerald has been awarded a 15% Gruyere increase and his own parking space. He does not have a car. He does not care.", Category: "mouse"},
	},
	// The Wedding Reception
	{
		{Text: "A wedding reception has been booked for tonight. The bride has requested 'all cheese, no exceptions.' We like her already.", Category: "review"},
		{Text: "The wedding cake has arrived. It is a tower of 14 different cheeses. The groom is crying. We think from joy.", Category: "cheese"},
		{Text: "The best man's speech includes the line 'their love is like a fine Comte - it only gets better with age.' Standing ovation.", Category: "review"},
		{Text: "UPDATE: The father of the bride has eaten an entire wheel of Camembert during his speech. Nobody stopped him. Nobody wanted to.", Category: "chef"},
		{Text: "The bouquet toss has been replaced with a cheese toss. Gerald caught it. He is now engaged to a block of Gruyere.", Category: "mouse"},
		{Text: "The wedding party has refused to leave. It is 2am. They have ordered another cheese board. We are so proud.", Category: "review"},
	},
	// The Power Cut
	{
		{Text: "CRITICAL: Power cut across all kitchens. The cheese cave temperature is rising. This is a Code Red.", Category: "alarm"},
		{Text: "Chef has declared a state of emergency. All staff are to fan the cheese with menus. Gerald is fanning with his tail.", Category: "mouse"},
		{Text: "The Camembert is showing signs of distress. It has begun to run. Not metaphorically. It is physically running off the shelf.", Category: "cheese"},
		{Text: "UPDATE: A generator has been found in the basement. It was being used to power a secret fondue station. We have questions.", Category: "mystery"},
		{Text: "Power has been restored. The cheese cave is secure. The Camembert has been recaptured. Gerald is receiving a medal for bravery.", Category: "mouse"},
	},
	// The Celebrity Visit
	{
		{Text: "A celebrity has been spotted at Table 1. We cannot say who for legal reasons. But they ordered three cheese boards.", Category: "review"},
		{Text: "The celebrity's bodyguard has asked if we can 'make the cheese quieter.' We do not know what this means.", Category: "mystery"},
		{Text: "UPDATE: The celebrity wants to meet Gerald. Gerald is in his dressing room (a shoebox behind the flour). He is not ready.", Category: "mouse"},
		{Text: "Gerald has emerged in a tiny bow tie. The celebrity is delighted. Photos have been taken. Gerald's agent will be in touch.", Category: "mouse"},
		{Text: "The celebrity has left a tip larger than our monthly cheese budget. We are building a statue in their honour. Out of cheese, obviously.", Category: "review"},
	},
	// The Apprentice's First Day
	{
		{Text: "New apprentice has arrived. He has asked where the microwave is. Chef's eye is twitching.", Category: "chef"},
		{Text: "The apprentice has been asked to grate parmesan. He is using a potato peeler. Nobody has the heart to tell him.", Category: "chef"},
		{Text: "UPDATE: The apprentice asked if Brie and Camembert are 'basically the same thing.' Three chefs have fainted.", Category: "alarm"},
		{Text: "The apprentice has put cheddar on the dessert menu. Surprisingly, Table 8 has ordered it. Twice.", Category: "review"},
		{Text: "End of Day 1. The apprentice has been assigned to Gerald for mentoring. Gerald has accepted on the condition of extra Gruyere.", Category: "mouse"},
	},
	// The Cheese Festival Prep
	{
		{Text: "Annual cheese festival is in 3 days. Chef has not slept in 48 hours. He is communicating exclusively in cheese names.", Category: "chef"},
		{Text: "UPDATE: Chef just said 'Roquefort' instead of 'good morning.' We think this means he's happy.", Category: "chef"},
		{Text: "The festival centrepiece - a 2-metre cheese sculpture of the restaurant - has collapsed. Chef is rebuilding it. Out of more cheese.", Category: "cheese"},
		{Text: "Gerald and his team have been put in charge of security. They have set up tiny checkpoints at all cave entrances.", Category: "mouse"},
		{Text: "Festival day. The queue is around the block. A man has been here since 4am. He is wearing a cape made of cheese cloth.", Category: "review"},
		{Text: "The festival is a triumph. We have sold 340kg of cheese in six hours. Chef has finally slept. He is dreaming about Gruyere.", Category: "chef"},
	},
	// The Haunted Cheese Cave
	{
		{Text: "Night security reports hearing whispers in the cheese cave. The words 'turn me over' were distinctly audible.", Category: "mystery"},
		{Text: "A paranormal investigation team has arrived. They have set up equipment around the Roquefort. Gerald is supervising.", Category: "mouse"},
		{Text: "The team's electromagnetic reader is going wild near the aged Stilton. The Stilton has always been dramatic.", Category: "mystery"},
		{Text: "UPDATE: The 'ghost' has been identified. It was Beatrice, the junior mouse, who has been living in the cave rent-free since Tuesday.", Category: "mouse"},
		{Text: "Beatrice has been evicted. She has filed a formal complaint. Gerald is representing her. He is charging his usual fee: one wheel of Brie.", Category: "mouse"},
	},
	// The Insurance Audit
	{
		{Text: "ALERT: Insurance auditor arriving at 14:00 to assess our cheese inventory. Current value: higher than the building.", Category: "alarm"},
		{Text: "The auditor has asked why we insure each wheel of Gruyere individually. Chef said 'because they're family.'", Category: "chef"},
		{Text: "UPDATE: The auditor has discovered that Gerald is listed as a 'cheese security consultant' on our payroll. He wants documentation.", Category: "mouse"},
		{Text: "Gerald has presented his employee ID badge. It has a photo. It is laminated. The auditor is questioning his life choices.", Category: "mouse"},
		{Text: "The audit is complete. We are now the most expensive cheese cave per square metre in the country. Chef is framing the certificate.", Category: "review"},
	},
}

// Standalone is the pool of one-off kitchen events, shuffled randomly
// between the sequence chains.
var Standalone = []Event{
	{Text: "Chef has eaten 4 slices of cheddar during prep. This is a new personal best.", Category: "chef"},
	{Text: "Customer asked if we have vegan cheese. The mice held an emergency meeting.", Category: "mouse"},
	{Text: "Someone left the raclette grill on overnight. Kitchen 3 now smells like a ski lodge.", Category: "alarm"},
	{Text: "The fondue pot has been declared a national treasure by table 7.", Category: "cheese"},
	{Text: "Intern asked what 'aged' means. Was given a 36-month Comte and told to sit quietly.", Category: "chef"},
	{Text: "The fridge thermometer is reading 'perfect cheese nap temperature.'", Category: "mystery"},
	{Text: "A customer tried to order a pizza without cheese. They have been politely escorted out.", Category: "review"},
	{Text: "Head chef found crying in the walk-in. Says the Stilton 'understands him.'", Category: "chef"},
	{Text: "The bread basket has filed a complaint about always being overshadowed by the cheese board.", Category: "mystery"},
	{Text: "Kitchen 1 reports a suspicious smell. Investigation reveals it's just the Epoisses doing its job.", Category: "cheese"},
	{Text: "A guest asked 'is this too much cheese?' The question has been forwarded to our legal team.", Category: "review"},
	{Text: "New hire asked where we keep the processed cheese. They have been let go.", Category: "chef"},
	{Text: "The grater in Kitchen 4 has been in continuous use for six hours. It is requesting a union.", Category: "mystery"},
	{Text: "A mouse was seen reading the wine list. Management is impressed by his initiative.", Category: "mouse"},
	{Text: "Health inspector arrived. Gerald hid behind the Roquefort. Bold choice.", Category: "mouse"},
	{Text: "Table 12 has sent back the cheese board. Reason: 'not enough cheese.' We respect this.", Category: "review"},
	{Text: "Someone wrote 'I LOVE CHEESE' in the condensation on the walk-in door. No suspects.", Category: "mystery"},
	{Text: "The sous chef has been caught dipping breadsticks directly into the fondue pot. Again.", Category: "chef"},
	{Text: "A customer has proposed to their partner using a cheese ring. They said yes.", Category: "review"},
	{Text: "The Brie has reached room temperature. A moment of silence for its perfection.", Category: "cheese"},
	{Text: "Someone has switched all the labels in the cheese cave. Chaos reigns.", Category: "mystery"},
	{Text: "Urgent: We are down to our last block of cheddar. This is not a drill.", Category: "cheese"},
	{Text: "The dishwasher has achieved sentience and is refusing to wash anything that touched blue cheese.", Category: "mystery"},
	{Text: "Gerald has been spotted teaching the new mice the layout of the cheese cave. Orientation day.", Category: "mouse"},
	{Text: "A guest left a review that just says 'cheese.' Five stars. We framed it.", Category: "review"},
	{Text: "Kitchen 2 temperature is rising. Unrelated: chef is arguing about whether Gouda is overrated.", Category: "chef"},
	{Text: "The cheese wire has snapped. Chef is attempting to slice Brie with a guitar string. Results pending.", Category: "chef"},
	{Text: "Delivery of 40kg of mozzarella has arrived. Kitchen staff are weeping with joy.", Category: "delivery"},
	{Text: "A child has asked if mice really eat cheese. Gerald has volunteered to demonstrate.", Category: "mouse"},
	{Text: "The halloumi is not melting. It never melts. It simply endures. We admire its commitment.", Category: "cheese"},
	{Text: "Table 3 is on their fourth cheese board. We are running out of crackers and dignity.", Category: "review"},
	{Text: "The night shift reports strange noises from the cheese cave. Investigation postponed until daylight.", Category: "mystery"},
	{Text: "Someone has drawn a face on the Edam. It looks disappointed. We understand.", Category: "mystery"},
	{Text: "The pasta machine has jammed. Investigation reveals a wedge of pecorino inside. No witnesses.", Category: "mystery"},
	{Text: "A pigeon has entered through the skylight and landed on the cheese board for Table 5. Table 5 is unbothered.", Category: "mystery"},
	{Text: "Chef has declared today 'Double Cheese Wednesday.' It is Friday. Nobody is correcting him.", Category: "chef"},
	{Text: "The freezer has been opened and closed 47 times today. The ice cream is fine. The mozzarella is traumatised.", Category: "cheese"},
	{Text: "An anonymous note has been left in the suggestion box: 'More cheese or I walk.' It is written in tiny handwriting.", Category: "mouse"},
	{Text: "Table 11 has asked for the cheese board 'but without the board.' We have served them a pile of cheese on the table. They are delighted.", Category: "review"},
	{Text: "The smoke detector in Kitchen 4 has been going off every 8 minutes. Chef says it's 'keeping time' for the risotto.", Category: "alarm"},
	{Text: "Gerald has been found asleep inside a hollowed-out Emmental. He says the holes 'provide ventilation.'", Category: "mouse"},
	{Text: "Two customers are arguing about whether cheddar or Gruyere is better on toast. Staff have started taking bets.", Category: "review"},
	{Text: "The cheese cave door was left open overnight. The Roquefort has made a break for it. It got as far as the hallway.", Category: "cheese"},
	{Text: "Accounting has sent an email asking why the cheese budget is 340% over forecast. Chef has replied with a photo of the cheese cave and the word 'beauty.'", Category: "chef"},
	{Text: "A customer has sent back the fondue saying it's 'too cheesy.' We are still processing this information.", Category: "review"},
	{Text: "The morning delivery includes a crate labelled 'DEFINITELY NOT MORE CHEESE.' It is more cheese. Chef is thrilled.", Category: "delivery"},
	{Text: "Gerald has started a cheese blog. First post: 'Top 10 Gruyeres I Have Personally Eaten, Ranked.' It has gone viral among the mice.", Category: "mouse"},
	{Text: "Kitchen 2 has run out of plates. They are now serving cheese on larger pieces of cheese. Guests think it's intentional.", Category: "chef"},
	{Text: "The walk-in fridge has been reorganised alphabetically. The Asiago is thrilled. The Wensleydale is furious.", Category: "cheese"},
	{Text: "A customer has asked for 'just a little cheese.' Chef is lying down in a dark room.", Category: "chef"},
	{Text: "The mice have started a book club. This month's selection: 'Who Moved My Cheese?' Gerald says it hits close to home.", Category: "mouse"},
	{Text: "Kitchen 3 reports that the fondue is 'looking at them.' Further investigation pending.", Category: "mystery"},
	{Text: "A wheel of Gouda has been found in the car park. It has a bus pass. We have questions.", Category: "mystery"},
	{Text: "The apprentice has asked if cheese grows on trees. He has been sent to the cheese cave to 'observe the harvest.'", Category: "chef"},
	{Text: "Table 14 has asked us to play 'happy birthday' on the cheese harp. We do not have a cheese harp. We are building one.", Category: "mystery"},
	{Text: "Gerald has submitted his expenses. He is claiming 14kg of Gruyere as 'professional development.'", Category: "mouse"},
	{Text: "The sourdough starter has become self-aware and is demanding to be paired with a specific Brie. We are complying.", Category: "mystery"},
	{Text: "Chef has been staring at the Comte for 20 minutes without blinking. He says he's 'communicating.' We believe him.", Category: "chef"},
	{Text: "A dog has entered the kitchen. It has ignored all the meat and gone straight for the cheese. Gerald approves.", Category: "mouse"},
	{Text: "The cheese delivery has arrived 6 hours early. The driver says he 'couldn't wait any longer.' We understand.", Category: "delivery"},
	{Text: "Table 2 has ordered a cheese board with 'extra board.' We have given them a plank. They are satisfied.", Category: "review"},
	{Text: "Someone has installed a tiny door at the base of the cheese cave wall. Gerald neither confirms nor denies involvement.", Category: "mouse"},
	{Text: "The Roquefort has been voted 'Employee of the Month.' It beat Gerald by one vote. Gerald is filing an appeal.", Category: "cheese"},
	{Text: "A customer has asked if we do takeaway. Chef has handed them a 3kg wheel of cheddar and said 'take it away.'", Category: "chef"},
	{Text: "The kitchen radio has been tuned to classical music. The Brie is noticeably calmer. The Stilton remains chaotic.", Category: "cheese"},
	{Text: "Night shift found a ransom note demanding 5kg of Gruyere for the safe return of the bread basket. Signed 'The Mice.'", Category: "mouse"},
	{Text: "A customer's child has drawn a picture of the restaurant. It is just cheese. We have put it on the fridge.", Category: "review"},
	{Text: "The Parmesan has been aged so long it now qualifies for a pension.", Category: "cheese"},
	{Text: "Gerald has been caught using the restaurant's WiFi to order Gruyere from a competitor. He shows no remorse.", Category: "mouse"},
	{Text: "A food blogger is photographing every dish from 17 angles. Other guests are growing restless. The cheese is photogenic.", Category: "review"},
	{Text: "The mop in Kitchen 1 smells like Camembert. We cannot determine if this is a problem or an achievement.", Category: "mystery"},
	{Text: "Chef has tattooed the molecular structure of cheddar on his forearm. HR has been notified. HR is impressed.", Category: "chef"},
	{Text: "A seagull has stolen an entire block of feta from the outdoor seating area. It is being hailed as a criminal mastermind.", Category: "mystery"},
	{Text: "The cheese grater has been named 'Grate Expectations.' Chef is very pleased with himself. Nobody else is.", Category: "chef"},
	{Text: "Beatrice has been promoted to Assistant Head of Bread. She is taking this very seriously. Gerald is mentoring her.", Category: "mouse"},
	{Text: "Table 6 has asked if they can move their table into the cheese cave. We are considering it.", Category: "review"},
	{Text: "The fire extinguisher in Kitchen 2 has been replaced with a fondue pot. Nobody knows when this happened.", Category: "alarm"},
	{Text: "A guest has written a love poem to our Gruyere on a napkin. We are having it professionally framed.", Category: "review"},
	{Text: "The dishwasher has refused to process any plate that didn't have cheese on it. Salad plates are piling up.", Category: "mystery"},
	{Text: "Gerald has learned to operate the till. He is ringing up extra Gruyere on every order. Sales are up 200%.", Category: "mouse"},
	{Text: "Chef has introduced a 'Cheese of the Day' feature. It has been cheddar for 47 consecutive days.", Category: "chef"},
	{Text: "A customer has asked for the bill in cheese. We have charged them 3 wheels of Gouda. They paid without hesitation.", Category: "review"},
}
