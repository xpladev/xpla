import hre from 'hardhat';
import {expect} from 'chai';

const { ethers } = await hre.network.connect();

describe('Bank', function () {
    it('query account balances', async function () {
        const bank = await ethers.getContractAt(
            'IBank',
            '0x1000000000000000000000000000000000000001'
        );
        const [signer] = await ethers.getSigners();
        const balance = await bank
            .getFunction('balance')
            .staticCall(signer.address, 'axpla');
        console.log('Balance:', balance);
        expect(balance).to.be.a('bigint');
        expect(balance).to.be.greaterThan(0);
    });

    it('query total supply', async function () {
        const bank = await ethers.getContractAt(
            'IBank',
            '0x1000000000000000000000000000000000000001'
        );

        const supply = await bank.getFunction('supplyOf').staticCall('axpla');
        console.log('Total supply:', supply);
        expect(supply).to.be.a('bigint');
        expect(supply).to.be.greaterThan(0);
    });
});