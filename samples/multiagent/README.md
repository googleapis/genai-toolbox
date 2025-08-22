# Multi-Agent Cymbal Travel Agency

> [!NOTE]
> This demo is NOT actively maintained.

## Introduction

This demo showcases a simplified travel agency called Cymbal Travel Agency. Check out our medium posting for more information.

Cymbal Travel Agency demonstrates how three agents can collaborate to research and book flights and hotels:

* **Customer Service Agent:** This agent is the primary interface for the customer. It receives travel requests, clarifies details, and relays informations to other agents.
* **Flight Agent:** This is a specialized agent that helps queries flight databases, list users' flight tickets and handles flight booking details.
* **Hotel Agent:** This is a specialized agent that helps searches for hotel accommodations, list users' hotel bookings, and manages hotel reservations.

![cymbal travel agency architecture](./architecture.png)

## Quickstart Demo

### Set up database

For this demo, I used an existing dataset from the [GenAI Databases Retrieval App v0.4.0](https://github.com/GoogleCloudPlatform/genai-databases-retrieval-app/tree/v0.4.0/data). For simpler set up, please follow the [README](https://github.com/GoogleCloudPlatform/genai-databases-retrieval-app/blob/v0.4.0/README.md) instructions and sets up via [run_database_init.py](https://github.com/GoogleCloudPlatform/genai-databases-retrieval-app/blob/v0.4.0/retrieval_service/run_database_init.py).

After parsing the dataset to your database, please ensure that your database consist of the following tables:

* airports
* amenities
* flights
* policies
* tickets

Next, we will generate some data for hotels:

<details>
<summary>SQL to create and insert hotels-related tables</summary>

```
CREATE TABLE hotels (
  name VARCHAR(255) NOT NULL,
  rating NUMERIC(2,1) NOT NULL,
  price INTEGER,
  city VARCHAR(255) NOT NULL
);

INSERT INTO hotels (name, rating, price, city) VALUES
    ('Rocky Mountain Retreat', 4, 285, 'Estes Park'),
    ('The Mile High Inn', 3, 210, 'Denver'),
    ('Aspen Creek Lodge', 5, 495, 'Aspen'),
    ('Breckenridge Vista', 4, 320, 'Breckenridge'),
    ('Garden of the Gods Resort', 5, 450, 'Colorado Springs'),
    ('Boulder Creek Hotel', 4, 250, 'Boulder'),
    ('The Vail Chalet', 5, 580, 'Vail'),
    ('Durango Junction Inn', 3, 185, 'Durango'),
    ('Union Station Hotel', 4, 350, 'Denver'),
    ('Telluride Mountain Suites', 5, 510, 'Telluride'),
    ('The Winter Park Lodge', 4, 290, 'Winter Park'),
    ('Steamboat Hot Springs', 3, 205, 'Steamboat Springs'),
    ('The Maroon Bells Inn', 4, 380, 'Aspen'),
    ('Crested Butte Getaway', 3, 240, 'Crested Butte'),
    ('The Denver Skyline', 4, 305, 'Denver'),
    ('Pikes Peak Inn', 3, 195, 'Colorado Springs'),
    ('Silverthorne Peaks Hotel', 4, 270, 'Silverthorne'),
    ('The Palisade Retreat', 5, 410, 'Palisade'),
    ('Grand Lake Lodge', 3, 230, 'Grand Lake'),
    ('Snowmass Village Resort', 5, 530, 'Snowmass Village'),
    ('The Copper Mountain Inn', 4, 315, 'Copper Mountain'),
    ('Keystone Lakeside Lodge', 3, 225, 'Keystone'),
    ('Arapahoe Basin Chalet', 4, 280, 'Dillon'),
    ('The Monarch Pass Lodge', 3, 190, 'Monarch'),
    ('Purgatory Pines Hotel', 4, 265, 'Durango'),
    ('The Aspen Peak Suites', 5, 520, 'Aspen'),
    ('Vail Village Hotel', 5, 610, 'Vail'),
    ('Steamboat River Inn', 4, 300, 'Steamboat Springs'),
    ('Telluride Grand Resort', 5, 550, 'Telluride'),
    ('Crested Butte Mountain Lodge', 4, 275, 'Crested Butte'),
    ('The Central Park Grand', 5, 650, 'Manhattan'),
    ('Brooklyn Bridge View', 4, 310, 'Brooklyn'),
    ('The Greenwich Village Inn', 3, 205, 'Manhattan'),
    ('Times Square Lights', 4, 380, 'Manhattan'),
    ('The Chelsea Art House', 4, 290, 'Manhattan'),
    ('Hotel Wall Street', 3, 230, 'Manhattan'),
    ('Queensboro River Hotel', 3, 185, 'Queens'),
    ('The NoMad Boutique', 5, 520, 'Manhattan'),
    ('The Harlem Jazz', 4, 240, 'Manhattan'),
    ('Staten Island Ferry Hotel', 3, 160, 'Staten Island'),
    ('The Upper East Side Manor', 5, 710, 'Manhattan'),
    ('The Broadway Performer', 4, 350, 'Manhattan'),
    ('The Plaza Tower', 5, 800, 'Manhattan'),
    ('Long Island City Loft', 3, 195, 'Queens'),
    ('The Battery Park Stay', 4, 270, 'Manhattan'),
    ('The SoHo Gallery', 5, 480, 'Manhattan'),
    ('The Bronx Botanical', 3, 170, 'The Bronx'),
    ('The Hudson Yards View', 4, 305, 'Manhattan'),
    ('The West Village Hideaway', 4, 330, 'Manhattan'),
    ('The Midtown Oasis', 5, 620, 'Manhattan');


CREATE TABLE
  "bookings" ( "user_id" TEXT,
    "user_name" TEXT,
    "user_email" TEXT,
    "hotel_name" TEXT,
    "hotel_city" TEXT,
    "hotel_rating" FLOAT,
    "hotel_total_price" FLOAT,
    "check_in_date" TEXT,
    "number_of_nights" INTEGER );
```

</details>

### Set up Toolbox

This demo utilizes [MCP Toolbox for Databases][https://github.com/googleapis/genai-toolbox]. Please follow the installation guidelines and install Toolbox locally.

Update the `tools.yaml` file with your database source information. For the simplicity of this demo, we did not utilize any Auth Services, hence, user-informations are all parsed automatically in `tools` with a `default` field.

Run Toolbox:

```
./toolbox
```

### Set up Cymbal Travel Agency application

1. [Install python][install-python] and set up a python [virtual environment][venv].

1. Install requirements:

    ```bash
    pip install -r requirements.txt
    ```

1. Run the application:

    ```bash
    python app.py
    ```

[install-python]: https://cloud.google.com/python/docs/setup#installing_python
[venv]: https://cloud.google.com/python/docs/setup#installing_and_using_virtualenv
